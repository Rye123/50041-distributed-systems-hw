package main

import (
	"fmt"
	"log"
	"slices"
	"sync"
	"time"
)

type CMState int
const (
	CM_BACKUP CMState = 0
	CM_PRIMARY CMState = 1
)

// Metadata for a particular page
type PageRecord struct {
	ownerId NodeId 
	copySet []NodeId // list of IDs of nodes with copies of this page
	pageId  PageId
	recordLock *sync.Mutex // locked while this is being modified
}

type CentralManager struct {
	cmId NodeId
	cmState CMState
	cmStateLock *sync.Mutex
	
	centralPort NodePort
	nodes map[NodeId]NodePort
	
	internalPort InternalPort // Port for communication with the other CM
	otherInternalPort InternalPort
	
	pageRecords map[PageId](*PageRecord)

	confirmations map[NodeId]Message // Confirmations from other nodes, reset prior to every request.e. This is all the CM needs to wait for.
	confirmationsLock *sync.Mutex

	electionOngoing bool
	electionId string
	electionNoRecvChan chan bool
	electionStateLock *sync.Mutex

	requestState RequestState
	requestId string
	requestPageId PageId
	requestStateLock *sync.Mutex

	counter int
	counterLock *sync.Mutex

	alive bool
	aliveStateLock *sync.Mutex
	exit chan bool
	
}

func NewCentralManager(cmId NodeId, centralPort NodePort, internalPort InternalPort, otherInternalPort InternalPort, nodePorts map[NodeId]NodePort, exit chan bool) *CentralManager {
	if cmId >= 0 {
		panic(fmt.Sprintf("Invalid CM ID %d.", cmId))
	}
	return &CentralManager{
		cmId, CM_BACKUP, &sync.Mutex{},
		centralPort, nodePorts, // Communication with Nodes
		internalPort, otherInternalPort, // Communication between CMs
		make(map[PageId](*PageRecord)),
		make(map[NodeId]Message), &sync.Mutex{},
		false, "", make(chan bool, 1), &sync.Mutex{},
		REQUEST_IDLE, "", InvalidPageId, &sync.Mutex{},
		0, &sync.Mutex{},
		true, &sync.Mutex{}, exit,
	}
}

func (cm *CentralManager) Reboot() {
	cm.aliveStateLock.Lock(); defer cm.aliveStateLock.Unlock()
	cm.confirmationsLock.Lock(); defer cm.confirmationsLock.Unlock()
	cm.counterLock.Lock(); defer cm.confirmationsLock.Unlock()
	cm.requestStateLock.Lock(); defer cm.requestStateLock.Unlock()
	cm.electionStateLock.Lock(); defer cm.electionStateLock.Unlock()
	cm.electionOngoing = false
	cm.electionId = ""
	cm.electionNoRecvChan = make(chan bool, 1)
	cm.requestState = REQUEST_IDLE
	cm.requestId = ""
	cm.requestPageId = InvalidPageId
	cm.counter = 0
	cm.confirmations = make(map[NodeId]Message)
	cm.pageRecords = make(map[PageId](*PageRecord))
	cm.alive = true
}

func (cm *CentralManager) Kill() {
	cm.aliveStateLock.Lock(); defer cm.aliveStateLock.Unlock()
	cm.alive = false
}

func (cm *CentralManager) Listen() {
	requestChan := make(chan Message)
	requesthandlerExit := make(chan bool)
	go cm.RequestHandler(requestChan, requesthandlerExit)
	
	//log.Printf("CM: Listening.")
	
	for {
		select {
		case msg, ok := <-cm.centralPort.RecvChan:
			cm.cmStateLock.Lock()
			cm.aliveStateLock.Lock()
			cmState := cm.cmState
			alive := cm.alive
			cm.aliveStateLock.Unlock()
			cm.cmStateLock.Unlock()
			if !ok {
				requesthandlerExit <- true
				log.Printf("CM: recvChan closed.")
				return
			}
			if !alive {
				// Drop message since we're DEAD
				continue
			}
			switch msg.MsgType {
			case MSG_RQ, MSG_WQ:
				if cmState == CM_BACKUP {
					panic(fmt.Sprintf("CM%d: Received MSG_RQ or MSG_WQ as Backup.", cm.cmId))
				}

				// Handover to RequestHandler
				msg := msg
				go func() { requestChan <- msg }()
			case MSG_RC, MSG_WC, MSG_IC:
				if cmState == CM_BACKUP {
					panic(fmt.Sprintf("CM%d: Received MSG_RC, MSG_WC or MSG_IC as Backup.", cm.cmId))
				}

				// Indicate to goroutine to carry out necessary operations
				cm.confirmationsLock.Lock()
				cm.confirmations[msg.SrcId] = msg
				cm.confirmationsLock.Unlock()
			case MSG_EL:
				// Start election
				go cm.startElection()
			default:
				panic(fmt.Sprintf("CM%d: Received unexpected message type %v from N%d with P%d.", cm.cmId, GetMessageType(msg.MsgType), msg.SrcId, msg.PageId))
			}
		case msg, ok := <-cm.internalPort.RecvChan:
			cm.cmStateLock.Lock()
			cm.aliveStateLock.Lock()
			cmState := cm.cmState
			alive := cm.alive
			cm.aliveStateLock.Unlock()
			cm.cmStateLock.Unlock()
			if !ok {
				requesthandlerExit <- true
				log.Printf("CM: recvChan closed.")
				return
			}
			if !alive {
				// Drop message since we're DEAD
				continue
			}

			switch msg.MsgType {
			case MSG_CM_ELECT_ME:
				//log.Printf("CM%d: Received ELECT_ME from other CM%d.", cm.cmId, msg.SrcId)
				if cmState == CM_PRIMARY {
					// Send ELECT_NO
					cm.sendElectionMsgToCM(msg.MsgId, MSG_CM_ELECT_NO)
				} else {
					// Here, we're also a backup.
					if cm.cmId > msg.SrcId {
						// Send ELECT_NO
						cm.sendElectionMsgToCM(msg.MsgId, MSG_CM_ELECT_NO)
					}
					// Otherwise, drop the message
				}
			case MSG_CM_ELECT_NO:
				cm.electionStateLock.Lock()
				if !cm.electionOngoing {
					panic(fmt.Sprintf("CM%d: Received ELECT_NO with no ongoing election,", cm.cmId))
				}
				if cm.electionId != msg.MsgId {
					panic(fmt.Sprintf("CM%d: Received ELECT_NO for %v, expected %v.", cm.cmId, msg.MsgId, cm.electionId))
				}
				cm.electionNoRecvChan <- true
				cm.electionStateLock.Unlock()
				
			case MSG_CM_RQ_RECV:
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received RQ_RECV as Primary.", cm.cmId))
				}
				
				cm.requestStateLock.Lock()
				if cm.requestState != REQUEST_IDLE {
					panic(fmt.Sprintf("CM%d: Received RQ_RECV from primary when currently handling another.", cm.cmId))
				}
				cm.requestId = msg.MsgId
				cm.requestState = REQUEST_RQ
				cm.requestPageId = msg.PageId
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.ensureRecordExists(pageId)
					if pageId == cm.requestPageId {
						cm.pageRecords[pageId].copySet = append(copyset, msg.SrcId)
					} else {
						cm.pageRecords[pageId].copySet = copyset
					}
				}

				for pageId, ownerId := range msg.SrcOwners {
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				cm.requestStateLock.Unlock()
			case MSG_CM_RC_RECV:
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received RC_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()
				if cm.requestState != REQUEST_RQ {
					panic(fmt.Sprintf("CM%d: Received RC_RECV from primary when currently NOT handling read request.", cm.cmId))
				}
				if cm.requestId != msg.MsgId {
					panic(fmt.Sprintf("CM%d: Received RC_RECV %v from primary, expected %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				cm.requestId = ""
				cm.requestState = REQUEST_IDLE
				cm.requestPageId = InvalidPageId
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].copySet = copyset
				}

				for pageId, ownerId := range msg.SrcOwners {
					cm.pageRecords[pageId].ownerId = ownerId
				}
				cm.requestStateLock.Unlock()
			case MSG_CM_WQ_RECV:
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received WQ_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()
				if cm.requestState != REQUEST_IDLE {
					panic(fmt.Sprintf("CM%d: Received WQ_RECV from primary when currently handling another.", cm.cmId))
				}
				cm.requestId = msg.MsgId
				cm.requestState = REQUEST_WQ
				cm.requestPageId = msg.PageId
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.ensureRecordExists(pageId)
					if pageId == cm.requestPageId {
						cm.pageRecords[pageId].copySet = append(copyset, msg.SrcId)
					} else {
						cm.pageRecords[pageId].copySet = copyset
					}
				}

				for pageId, ownerId := range msg.SrcOwners {
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				cm.requestStateLock.Unlock()
			case MSG_CM_WC_RECV:
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received WC_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()
				if cm.requestState != REQUEST_WQ {
					panic(fmt.Sprintf("CM%d: Received WC_RECV from primary when currently NOT handling write request.", cm.cmId))
				}
				if cm.requestId != msg.MsgId {
					panic(fmt.Sprintf("CM%d: Received WC_RECV %v from primary, expected %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				cm.requestId = ""
				cm.requestState = REQUEST_IDLE
				cm.requestPageId = InvalidPageId
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.ensureRecordExists(pageId)
					// This should be the empty copyset
					cm.pageRecords[pageId].copySet = copyset
				}

				for pageId, ownerId := range msg.SrcOwners {
					cm.pageRecords[pageId].ownerId = ownerId
				}
				cm.requestStateLock.Unlock()
				
			}
		case <-cm.exit:
			log.Printf("CM: Exit.")
			requesthandlerExit <- true
			return
		}
	}
}

func (cm *CentralManager) startElection() {
	//log.Printf("CM%d: Starting election..", cm.cmId)
	electionWin := false
	
	if cm.cmState == CM_BACKUP {
		cm.electionStateLock.Lock()
		if cm.electionOngoing {
			// Drop message if we alr are waiting for an election
			cm.electionStateLock.Unlock()
			return
		}

		cm.electionOngoing = true
		cm.electionNoRecvChan = make(chan bool, 1) // TODO: address case where message arrives late?
		cm.electionId = cm.generateElectionId()
		
		// Send ELECT_ME to other CM
		cm.sendElectionMsgToCM(cm.electionId, MSG_CM_ELECT_ME)
		cm.electionStateLock.Unlock()

		// Block until timeout, or ELECT_NO
		timer := time.NewTimer(TIMEOUT_INTV)

		select {
		case <-cm.electionNoRecvChan:
			timer.Stop()
			electionWin = false
		case <-timer.C:
			// Timeout
			electionWin = true
		}
	} else {
		// If we're a primary, we win by default
		electionWin = true
	}

	// Broadcast victory
	if electionWin {
		cm.cmStateLock.Lock()
		cm.cmState = CM_PRIMARY
		cm.cmStateLock.Unlock()
		//log.Printf("CM%d: Won election. Broadcasting win.", cm.cmId)
		for nodeId := range cm.nodes {
			log.Printf("CM%d: Sent win to N%d.", cm.cmId, nodeId)
			cm.electionStateLock.Lock()
			if cm.electionId == "" {
				cm.electionId = cm.generateElectionId()
			}
			nodeId := nodeId
			cm.send(cm.electionId, MSG_ELWIN, cm.cmId, nodeId, InvalidPageId)
			cm.electionStateLock.Unlock()
		}
	}

	// Clear election
	cm.electionStateLock.Lock()
	cm.electionId = ""
	cm.electionOngoing = false
	cm.electionStateLock.Unlock()
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
				cm.recvReadRequest(req.SrcId, req.PageId, req.MsgId)
			case MSG_WQ:
				cm.recvWriteRequest(req.SrcId, req.PageId, req.MsgId)
			default:
				panic(fmt.Sprintf("CM.RequestHandler: Unexpected msgType %v", GetMessageType(req.MsgType)))
			}
		case <-exitChan:
			log.Printf("CM: RequestHandler stopped.")
			return
		}
	}
}

func (cm *CentralManager) recvReadRequest(rdrId NodeId, pageId PageId, reqId string) {
	log.Printf("CM%d: Received RQ(N%d, P%d)", cm.cmId, rdrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If currently processing another request, drop this.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_IDLE && cm.requestId != reqId {
		log.Printf("CM%d: Dropped RQ(N%d, P%d, %v), currently handling request %v.", cm.cmId, rdrId, pageId, reqId, cm.requestId)
		cm.requestStateLock.Unlock()
		return
	}
	cm.requestState = REQUEST_RQ
	cm.requestId = reqId
	cm.requestPageId = pageId
	cm.requestStateLock.Unlock()

	// Forward request to replica
	cm.sendUpdateMsgToCM(reqId, MSG_CM_RQ_RECV, rdrId, pageId)

	// reset confirmations
	cm.confirmationsLock.Lock()
	cm.confirmations = make(map[NodeId]Message)
	cm.confirmationsLock.Unlock()

	if rdrId == cm.pageRecords[pageId].ownerId { panic(fmt.Sprintf("CM%d: Owner N%d requested to read own page", cm.cmId, rdrId))}
	
	// 1. Add reader to copyset
	cm.addToCopySet(rdrId, pageId)

	// 2. Send read forward
	ownerId := cm.pageRecords[pageId].ownerId
	if ownerId == InvalidNodeId {
		panic(fmt.Sprintf("CM%d: Non-existent owner for page P%d.", cm.cmId, pageId))
	}
	cm.send(reqId, MSG_RF, rdrId, ownerId, pageId)

	//TODO: HOW to get this part, if backup takes over at this point?
	
	// 3. Block until we receive RC from reader
	rc_rcvd := false
	for !rc_rcvd {
		// Exit if node is killed
		cm.aliveStateLock.Lock()
		if !cm.alive {
			// Drop message since we're DEAD
			cm.aliveStateLock.Unlock()
			return
		}
		cm.aliveStateLock.Unlock()
		
		// Check for RC
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

	// 4. Send RC_RECV to replica, send RC_ACK to node
	cm.sendUpdateMsgToCM(reqId, MSG_CM_RC_RECV, rdrId, pageId)
	cm.send(reqId, MSG_RC_ACK, cm.cmId, rdrId, pageId)
	log.Printf("CM%d: Completed RQ(N%d, P%d)", cm.cmId, rdrId, pageId)
	cm.requestStateLock.Lock()
	cm.requestId = ""
	cm.requestState = REQUEST_IDLE
	cm.requestPageId = InvalidPageId
	cm.requestStateLock.Unlock()
}

func (cm *CentralManager) recvWriteRequest(wtrId NodeId, pageId PageId, reqId string) {
	log.Printf("CM%d: Received WQ(N%d, P%d)", cm.cmId, wtrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If currently processing another request, drop this.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_IDLE && cm.requestId != reqId {
		log.Printf("CM%d: Dropped WQ(N%d, P%d, %v), currently handling request %v.", cm.cmId, wtrId, pageId, reqId, cm.requestId)
		cm.requestStateLock.Unlock()
		return
	}
	cm.requestState = REQUEST_WQ
	cm.requestId = reqId
	cm.requestPageId = pageId
	cm.requestStateLock.Unlock()

	// Forward request to replica
	cm.sendUpdateMsgToCM(reqId, MSG_CM_WQ_RECV, wtrId, pageId)

	// reset confirmations
	cm.confirmationsLock.Lock()
	cm.confirmations = make(map[NodeId]Message)
	cm.confirmationsLock.Unlock()

	// 1. Invalidate all copies
	for _, copyId := range cm.pageRecords[pageId].copySet {
		cm.send(reqId, MSG_IV, wtrId, copyId, pageId)
	}

	// 2. Block until we receive IC from all copies
	expected_ics := len(cm.pageRecords[pageId].copySet) // shouldn't change because it's locked
	ic_rcvd := 0
	for ic_rcvd < expected_ics {
		// Exit if node is killed
		cm.aliveStateLock.Lock()
		if !cm.alive {
			// Drop message since we're DEAD
			cm.aliveStateLock.Unlock()
			return
		}
		cm.aliveStateLock.Unlock()

		// Check for IC
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
		cm.send(reqId, MSG_WI, cm.cmId, wtrId, pageId)
		log.Printf("CM: Initialised NEW page P%d with owner N%d.", pageId, wtrId)
	} else {
		cm.send(reqId, MSG_WF, wtrId, ownerId, pageId)
	}

	// 4. Block until we receive WC from writer
	wc_rcvd := false
	for !wc_rcvd {
		// Exit if node is killed
		cm.aliveStateLock.Lock()
		if !cm.alive {
			// Drop message since we're DEAD
			cm.aliveStateLock.Unlock()
			return
		}
		cm.aliveStateLock.Unlock()

		// Check for WC
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

	// 6. Send WC_RECV to replica, send WC_ACK to node
	cm.sendUpdateMsgToCM(reqId, MSG_CM_WC_RECV, wtrId, pageId)
	cm.send(reqId, MSG_WC_ACK, cm.cmId, wtrId, pageId)
	
	log.Printf("CM: Completed WQ(N%d, P%d)", wtrId, pageId)
	cm.requestStateLock.Lock()
	cm.requestId = ""
	cm.requestState = REQUEST_IDLE
	cm.requestPageId = InvalidPageId
	cm.requestStateLock.Unlock()
	
}

func (cm *CentralManager) send(msgId string, msgType MsgType, srcId, dstId NodeId, pageId PageId) {
	msg := NewMessage(msgId, msgType, srcId, pageId)
	cm.nodes[dstId].RecvChan <- msg
	//log.Printf("CM: Send %v for P%d from N%d to N%d.", GetMessageType(msgType), pageId, srcId, dstId)
}

func (cm *CentralManager) sendElectionMsgToCM(electionId string, msgType MsgType) {
	msg := NewCMElectionMessage(electionId, msgType, cm.cmId)
	cm.otherInternalPort.RecvChan <- msg
}

func (cm *CentralManager) sendUpdateMsgToCM(reqId string, msgType MsgType, nodeId NodeId, pageId PageId) {
	copysets := make(map[PageId]([]NodeId))
	owners := make(map[PageId]NodeId)
	for pageId, record := range cm.pageRecords {
		copysets[pageId] = make([]NodeId, 0)
		for _, nodeId := range record.copySet {
			copysets[pageId] = append(copysets[pageId], nodeId)
		}
		owners[pageId] = record.ownerId
	}
	msg := NewCMUpdate(reqId, msgType, nodeId, pageId, copysets, owners)
	cm.otherInternalPort.RecvChan <- msg
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

func (cm *CentralManager) generateElectionId() string {
	cm.counterLock.Lock(); defer cm.counterLock.Unlock()
	elId := fmt.Sprintf("EL_N%d_%d", cm.cmId, cm.counter)
	cm.counter++
	return elId
}
