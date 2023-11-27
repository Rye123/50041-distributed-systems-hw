package main

import (
	"fmt"
	"log"
	"slices"
	"sync"
)

// Metadata for a particular page
type PageRecord struct {
	ownerId NodeId 
	copySet []NodeId // list of IDs of nodes with copies of this page
	pageId  PageId
	recordLock *sync.Mutex // locked while this is being modified
}

type CentralManager struct {
	centralPort NodePort
	
	nodes map[NodeId]NodePort
	pageRecords map[PageId](*PageRecord)

	confirmations map[NodeId]Message // Confirmations from other nodes, reset prior to every request.e. This is all the CM needs to wait for.
	confirmationsLock *sync.Mutex

	exit chan bool
	
}

func NewCentralManager(centralPort NodePort, nodePorts map[NodeId]NodePort, exit chan bool) *CentralManager {
	return &CentralManager{
		centralPort, nodePorts,
		make(map[PageId](*PageRecord)),
		make(map[NodeId]Message), &sync.Mutex{},
		exit,
	}
}

func (cm *CentralManager) Listen() {
	requestChan := make(chan Message)
	requesthandlerExit := make(chan bool)
	go cm.RequestHandler(requestChan, requesthandlerExit)
	
	//log.Printf("CM: Listening.")
	
	for {
		select {
		case msg, ok := <-cm.centralPort.RecvChan:
			if !ok {
				requesthandlerExit <- true
				log.Printf("CM: recvChan closed.")
				return
			}

			switch msg.MsgType {
			case MSG_RQ, MSG_WQ:
				msg := msg
				go func() { requestChan <- msg }()
			case MSG_RC, MSG_WC, MSG_IC:
				cm.confirmationsLock.Lock()
				cm.confirmations[msg.SrcId] = msg
				cm.confirmationsLock.Unlock()
			default:
				panic(fmt.Sprintf("CM: Received unexpected message type %v from N%d with P%d.", GetMessageType(msg.MsgType), msg.SrcId, msg.PageId))
			}
		case <-cm.exit:
			log.Printf("CM: Exit.")
			requesthandlerExit <- true
			return
		}
	}
}

// A sequential handler for requests
func (cm *CentralManager) RequestHandler(requestChan chan Message, exitChan chan bool) {
	for {
		select {
		case req, ok := <-requestChan:
			if !ok {
				log.Printf("CM: RequestHandler channel closed.")
				return
			}
			switch req.MsgType {
			case MSG_RQ:
				cm.recvReadRequest(req.SrcId, req.PageId)
			case MSG_WQ:
				cm.recvWriteRequest(req.SrcId, req.PageId)
			default:
				panic(fmt.Sprintf("CM.RequestHandler: Unexpected msgType %v", GetMessageType(req.MsgType)))
			}
		case <-exitChan:
			log.Printf("CM: RequestHandler stopped.")
			return
		}
	}
}

func (cm *CentralManager) recvReadRequest(rdrId NodeId, pageId PageId) {
	log.Printf("CM: Received RQ(N%d, P%d)", rdrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// reset confirmations
	cm.confirmationsLock.Lock()
	cm.confirmations = make(map[NodeId]Message)
	cm.confirmationsLock.Unlock()

	if rdrId == cm.pageRecords[pageId].ownerId { panic(fmt.Sprintf("CM: Owner N%d requested to read own page", rdrId))}
	
	// 1. Add reader to copyset
	cm.addToCopySet(rdrId, pageId)

	// 2. Send read forward
	ownerId := cm.pageRecords[pageId].ownerId
	if ownerId == InvalidNodeId {
		panic(fmt.Sprintf("CM: Non-existent owner for page P%d.", pageId))
	}
	cm.send(MSG_RF, rdrId, ownerId, pageId)
	
	// 3. Block until we receive RC from reader
	rc_rcvd := false
	for !rc_rcvd {
		cm.confirmationsLock.Lock()
		if msg, ok := cm.confirmations[rdrId]; ok {
			if msg.MsgType == MSG_RC {
				// if we've received a READ CONFIRM from reader
				cm.confirmationsLock.Unlock()
				rc_rcvd = true
				break
			} else {
				panic(fmt.Sprintf("Received non-RC from N%d: %v", rdrId, GetMessageType(msg.MsgType)))
			}
		}
		cm.confirmationsLock.Unlock()
	}
	log.Printf("CM: Completed RQ(N%d, P%d)", rdrId, pageId)
}

func (cm *CentralManager) recvWriteRequest(wtrId NodeId, pageId PageId) {
	log.Printf("CM: Received WQ(N%d, P%d)", wtrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// reset confirmations
	cm.confirmationsLock.Lock()
	cm.confirmations = make(map[NodeId]Message)
	cm.confirmationsLock.Unlock()

	// 1. Invalidate all copies
	for _, copyId := range cm.pageRecords[pageId].copySet {
		cm.send(MSG_IV, wtrId, copyId, pageId)
	}

	// 2. Block until we receive IC from all copies
	expected_ics := len(cm.pageRecords[pageId].copySet) // shouldn't change because it's locked
	ic_rcvd := 0
	for ic_rcvd < expected_ics {
		ic_rcvd = 0
		cm.confirmationsLock.Lock()
		for _, msg := range cm.confirmations {
			if msg.MsgType == MSG_IC { ic_rcvd++ }
		}

		if ic_rcvd > expected_ics {
			panic(fmt.Sprintf("CM: Received %d ICs, expected %d", ic_rcvd, expected_ics))
		}
		cm.confirmationsLock.Unlock()
	}

	// 3. Send write forward to owner
	ownerId := cm.pageRecords[pageId].ownerId
	if ownerId == InvalidNodeId {
		// Send the page back with a write init (WI), since node expects a WP or WI.
		cm.send(MSG_WI, InvalidNodeId, wtrId, pageId)
		log.Printf("CM: Initialised NEW page P%d with owner N%d.", pageId, wtrId)
	} else {
		cm.send(MSG_WF, wtrId, ownerId, pageId)
	}

	// 4. Block until we receive WC from writer
	wc_rcvd := false
	for !wc_rcvd {
		cm.confirmationsLock.Lock()
		if msg, ok := cm.confirmations[wtrId]; ok {
			if msg.MsgType == MSG_WC {
				// if we've received a WRITE CONFIRM from reader
				wc_rcvd = true
			}
		}
		cm.confirmationsLock.Unlock()
	}

	// 5. Set new owner
	cm.pageRecords[pageId].ownerId = wtrId
	
	log.Printf("CM: Completed WQ(N%d, P%d)", wtrId, pageId)
	
}

func (cm *CentralManager) send(msgType MsgType, srcId, dstId NodeId, pageId PageId) {
	msg := NewMessage(msgType, srcId, pageId)
	cm.nodes[dstId].RecvChan <- msg
	//log.Printf("CM: Send %v for P%d from N%d to N%d.", GetMessageType(msgType), pageId, srcId, dstId)
}
func (cm *CentralManager) ensureRecordExists(pageId PageId) {
	if _, ok := cm.pageRecords[pageId]; !ok {
		cm.pageRecords[pageId] = &PageRecord{InvalidNodeId, make([]NodeId, 0), pageId, &sync.Mutex{}}
	}
}

func (cm *CentralManager) addToCopySet(tgtId NodeId, pageId PageId) {
	if slices.ContainsFunc(cm.pageRecords[pageId].copySet, func(nodeId NodeId) bool {return nodeId == tgtId}) {
		// tgtId is already in CopySet
		return
	}
	cm.pageRecords[pageId].copySet = append(cm.pageRecords[pageId].copySet, tgtId)
}

func (cm *CentralManager) removeFromCopySet(tgtId NodeId, pageId PageId) {
	cm.pageRecords[pageId].copySet = slices.DeleteFunc(cm.pageRecords[pageId].copySet, func(nodeId NodeId) bool {
		return nodeId == tgtId // Delete this ID
	})
}
