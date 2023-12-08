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

	invalCount int
	invalExpected int
	invalLock *sync.Mutex

	electionOngoing bool
	electionId string
	electionNoRecvChan chan bool
	electionStateLock *sync.Mutex

	requestState RequestState
	requestorId NodeId
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
		0, 0, &sync.Mutex{},
		false, "", make(chan bool, 1), &sync.Mutex{},
		REQUEST_IDLE, InvalidNodeId, "", InvalidPageId, &sync.Mutex{},
		0, &sync.Mutex{},
		true, &sync.Mutex{}, exit,
	}
}

func (cm *CentralManager) Reboot() {
	cm.aliveStateLock.Lock(); defer cm.aliveStateLock.Unlock()
	cm.invalLock.Lock(); defer cm.invalLock.Unlock()
	cm.counterLock.Lock(); defer cm.counterLock.Unlock()
	cm.requestStateLock.Lock(); defer cm.requestStateLock.Unlock()
	cm.cmStateLock.Lock(); defer cm.cmStateLock.Unlock()
	cm.electionStateLock.Lock()
	cm.electionOngoing = false
	cm.electionId = ""
	cm.electionNoRecvChan = make(chan bool, 1)
	cm.electionStateLock.Unlock()
	cm.requestState = REQUEST_IDLE
	cm.requestId = ""
	cm.requestorId = InvalidNodeId
	cm.requestPageId = InvalidPageId
	cm.counter = 0
	cm.invalCount = 0
	cm.invalExpected = 0
	cm.pageRecords = make(map[PageId](*PageRecord))
	cm.alive = true
	cm.cmState = CM_BACKUP
	go cm.startElection()

	// Don't return until election is over
	for {
		cm.electionStateLock.Lock()
		if !cm.electionOngoing {
			cm.electionStateLock.Unlock()
			break
		}
		cm.electionStateLock.Unlock()
	}
	log.Printf("CM%d: Rebooted.", cm.cmId)
}

func (cm *CentralManager) Kill() {
	cm.aliveStateLock.Lock(); defer cm.aliveStateLock.Unlock()
	cm.alive = false
	log.Printf("CM%d: Killed.", cm.cmId)
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
			case MSG_IC:
				if cmState == CM_BACKUP {
					panic(fmt.Sprintf("CM%d: Received MSG_IC as Backup.", cm.cmId))
				}

				cm.recvInvalConfirm(msg.SrcId, msg.PageId, msg.MsgId)

			case MSG_RC:
				if cmState == CM_BACKUP {
					panic(fmt.Sprintf("CM%d: Received MSG_RC as Backup.", cm.cmId))
				}

				cm.recvReadConfirm(msg.SrcId, msg.PageId, msg.MsgId)
				
			case MSG_WC:
				if cmState == CM_BACKUP {
					panic(fmt.Sprintf("CM%d: Received MSG_WC as Backup.", cm.cmId))
				}

				cm.recvWriteConfirm(msg.SrcId, msg.PageId, msg.MsgId)
				
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
					cm.sendEmptyUpdateMsgToCM()
					cm.sendElectionMsgToCM(msg.MsgId, MSG_CM_ELECT_NO)
				} else {
					// Here, we're also a backup.
					if cm.cmId > msg.SrcId {
						// Send ELECT_NO
						cm.sendEmptyUpdateMsgToCM()
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

			case MSG_CM_UPDATE:
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received CM_UPDATE as Primary.", cm.cmId))
				}
				log.Printf("CM%d: Updated.", cm.cmId)

				for pageId, ownerId := range msg.SrcOwners {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.pageRecords[pageId].copySet = copyset
				}
				
			case MSG_CM_RQ_RECV:
				// Received RQ_RECV.
				// Possible cases: Backup is in REQUEST_IDLE.
				
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received RQ_RECV as Primary.", cm.cmId))
				}
				
				cm.requestStateLock.Lock()
				if cm.requestState != REQUEST_IDLE {
					// Not possible!
					panic(fmt.Sprintf("CM%d: Received RQ_RECV (%v) from primary when NOT idle (handling %v).", cm.cmId, msg.MsgId, cm.requestId))
				}

				cm.requestId = msg.MsgId
				cm.requestState = REQUEST_RQ
				cm.requestPageId = msg.PageId
				cm.requestorId = msg.SrcId // Update msgs have a srcId of the node

				for pageId, ownerId := range msg.SrcOwners {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.pageRecords[pageId].copySet = copyset
				}
				
				cm.requestStateLock.Unlock()
			case MSG_CM_RC_RECV:
				// Received RC_RECV.
				// Possible cases: Backup is in REQUEST_IDLE (just rebooted), or in REQUEST_RQ (saw the earlier RQ)
				
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received RC_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()
				if cm.requestState == REQUEST_WQ {
					// Not possible!
					panic(fmt.Sprintf("CM%d: Received RC_RECV (%v) from primary when currently handling WQ %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				if cm.requestState == REQUEST_RQ && cm.requestId != msg.MsgId {
					// Not possible: We already recorded a request, but suddenly primary sends us a confirmation for another?
					panic(fmt.Sprintf("CM%d: Received RC_RECV (%v) from primary when expecting RC %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				cm.requestId = ""
				cm.requestState = REQUEST_IDLE
				cm.requestPageId = InvalidPageId
				cm.requestorId = InvalidNodeId

				for pageId, ownerId := range msg.SrcOwners {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.pageRecords[pageId].copySet = copyset
				}
				
				cm.requestStateLock.Unlock()
			case MSG_CM_WQ_RECV:
				// Received WQ_RECV.
				// Possible cases: Backup is in REQUEST_IDLE.
				
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received WQ_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()

				if cm.requestState != REQUEST_IDLE {
					// Not possible!
					panic(fmt.Sprintf("CM%d: Received WQ_RECV (%v) from primary when NOT idle (handling %v).", cm.cmId, msg.MsgId, cm.requestId))
				}
				
				cm.requestId = msg.MsgId
				cm.requestState = REQUEST_WQ
				cm.requestPageId = msg.PageId
				cm.requestorId = msg.SrcId // Update msgs have a srcId of the node

				for pageId, ownerId := range msg.SrcOwners {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.pageRecords[pageId].copySet = copyset
				}
				
				cm.requestStateLock.Unlock()
			case MSG_CM_WC_RECV:
				// Received WC_RECV.
				// Possible cases: Backup is in REQUEST_IDLE (just rebooted), or in REQUEST_WQ (saw the earlier WQ)
				
				if cmState == CM_PRIMARY {
					panic(fmt.Sprintf("CM%d: Received WC_RECV as Primary.", cm.cmId))
				}
				cm.requestStateLock.Lock()
				if cm.requestState == REQUEST_RQ {
					// Not possible!
					panic(fmt.Sprintf("CM%d: Received WC_RECV (%v) from primary when currently handling RQ %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				if cm.requestState == REQUEST_WQ && cm.requestId != msg.MsgId {
					// Not possible: We already recorded a request, but suddenly primary sends us a confirmation for another?
					panic(fmt.Sprintf("CM%d: Received WC_RECV (%v) from primary when expecting WC %v.", cm.cmId, msg.MsgId, cm.requestId))
				}

				cm.requestId = ""
				cm.requestState = REQUEST_IDLE
				cm.requestPageId = InvalidPageId
				cm.requestorId = InvalidNodeId

				for pageId, ownerId := range msg.SrcOwners {
					cm.ensureRecordExists(pageId)
					cm.pageRecords[pageId].ownerId = ownerId
				}
				
				for pageId, copyset := range msg.SrcCopysets {
					cm.pageRecords[pageId].copySet = copyset
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
		log.Printf("CM%d: Won election.", cm.cmId)
		for nodeId := range cm.nodes {
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
	log.Printf("CM%d: Received RQ(N%d, P%d, %v)", cm.cmId, rdrId, pageId, reqId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If currently processing another request, drop this.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_IDLE {
		if cm.requestState == REQUEST_RQ && cm.requestId == reqId {
			log.Printf("CM%d: Received retransmission of request %v. Handling.", cm.cmId, reqId)
		} else {
			log.Printf("CM%d: Dropped RQ(N%d, P%d, %v), currently handling request %v.", cm.cmId, rdrId, pageId, reqId, cm.requestId)
			cm.requestStateLock.Unlock()
			return
		}
	}
	cm.requestState = REQUEST_RQ
	cm.requestorId = rdrId
	cm.requestId = reqId
	cm.requestPageId = pageId
	cm.requestStateLock.Unlock()

	// Forward request to replica
	cm.sendUpdateMsgToCM(reqId, MSG_CM_RQ_RECV, rdrId, pageId)
	if rdrId == cm.pageRecords[pageId].ownerId { panic(fmt.Sprintf("CM%d: Owner N%d requested to read own page", cm.cmId, rdrId))}
	
	// 1. Add reader to copyset
	cm.addToCopySet(rdrId, pageId)

	// 2. Send read forward
	ownerId := cm.pageRecords[pageId].ownerId
	if ownerId == InvalidNodeId {
		panic(fmt.Sprintf("CM%d: Non-existent owner for page P%d.", cm.cmId, pageId))
	}
	cm.send(reqId, MSG_RF, rdrId, ownerId, pageId)
}

func (cm *CentralManager) recvReadConfirm(rdrId NodeId, pageId PageId, reqId string) {
	log.Printf("CM%d: Received RC(N%d, P%d)", cm.cmId, rdrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If not processing a request, drop the RC.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_RQ ||  cm.requestId != reqId {
		log.Printf("CM%d: Dropped RC(N%d, P%d, %v), currently handling request %v.", cm.cmId, rdrId, pageId, reqId, cm.requestId)
		cm.requestStateLock.Unlock()
		return
	}
	cm.requestStateLock.Unlock()

	// Send RC_RECV to replica, send RC_ACK to node
	cm.sendUpdateMsgToCM(reqId, MSG_CM_RC_RECV, rdrId, pageId)
	cm.send(reqId, MSG_RC_ACK, cm.cmId, rdrId, pageId)
	log.Printf("CM%d: Completed RQ(N%d, P%d, %v)", cm.cmId, rdrId, pageId, reqId)
	cm.requestStateLock.Lock()
	cm.requestorId = InvalidNodeId
	cm.requestId = ""
	cm.requestState = REQUEST_IDLE
	cm.requestPageId = InvalidPageId
	cm.requestStateLock.Unlock()
}

func (cm *CentralManager) recvWriteRequest(wtrId NodeId, pageId PageId, reqId string) {
	log.Printf("CM%d: Received WQ(N%d, P%d, %v)", cm.cmId, wtrId, pageId, reqId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If currently processing another request, drop this.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_IDLE {
		if cm.requestState == REQUEST_WQ && cm.requestId == reqId {
			log.Printf("CM%d: Received retransmission of request %v. Handling.", cm.cmId, reqId)
		} else {
			log.Printf("CM%d: Dropped WQ(N%d, P%d, %v), currently handling request %v.", cm.cmId, wtrId, pageId, reqId, cm.requestId)
			cm.requestStateLock.Unlock()
			return
		}
	}
	cm.requestState = REQUEST_WQ
	cm.requestorId = wtrId
	cm.requestId = reqId
	cm.requestPageId = pageId
	cm.requestStateLock.Unlock()

	// Forward request to replica
	cm.sendUpdateMsgToCM(reqId, MSG_CM_WQ_RECV, wtrId, pageId)

	if len(cm.pageRecords[pageId].copySet) == 0 {
		// No copies in copy set, skip to write forward/init
		log.Printf("CM%d: No copies. Sending write.", cm.cmId)
		ownerId := cm.pageRecords[pageId].ownerId
		if ownerId == InvalidNodeId {
			// Send the page back with a write init (WI), since node expects a WP or WI.
			cm.send(reqId, MSG_WI, cm.cmId, cm.requestorId, pageId)
			log.Printf("CM: Initialised NEW page P%d with owner N%d.", pageId, cm.requestorId)
		} else {
			cm.send(reqId, MSG_WF, cm.requestorId, ownerId, pageId)
		}
		log.Printf("CM%d: Sent write forward/init.", cm.cmId)
		return
	}

	// Reset invals
	cm.invalLock.Lock()
	cm.invalCount = 0
	cm.invalExpected = len(cm.pageRecords[pageId].copySet)
	cm.invalLock.Unlock()

	// 1. Invalidate all copies
	for _, copyId := range cm.pageRecords[pageId].copySet {
		if cm.pageRecords[pageId].ownerId == copyId {
			panic(fmt.Sprintf("CM%d: N%d appears in copyset, but is the owner", cm.cmId, copyId))
		}
		cm.send(reqId, MSG_IV, wtrId, copyId, pageId)
	}

}

func (cm *CentralManager) recvInvalConfirm(invId NodeId, pageId PageId, reqId string) {
	//log.Printf("CM%d: Received IC(N%d, P%d, %v)", cm.cmId, invId, pageId, reqId)

	cm.invalLock.Lock(); defer cm.invalLock.Unlock()
	cm.invalCount++

	if cm.invalCount > cm.invalExpected {
		panic(fmt.Sprintf("CM%d: Received %d ICs, expected %d.", cm.cmId, cm.invalCount, cm.invalExpected))
	} else if cm.invalCount < cm.invalExpected {
		return
	}
	
	log.Printf("CM%d: Invalidated copies. Sending write.", cm.cmId)

	// Send write forward to owner
	ownerId := cm.pageRecords[pageId].ownerId
	if ownerId == InvalidNodeId {
		// Send the page back with a write init (WI), since node expects a WP or WI.
		cm.send(reqId, MSG_WI, cm.cmId, cm.requestorId, pageId)
		log.Printf("CM: Initialised NEW page P%d with owner N%d.", pageId, cm.requestorId)
	} else {
		cm.send(reqId, MSG_WF, cm.requestorId, ownerId, pageId)
	}
	log.Printf("CM%d: Sent write forward/init.", cm.cmId)
}

func (cm *CentralManager) recvWriteConfirm(wtrId NodeId, pageId PageId, reqId string) {
	log.Printf("CM%d: Received WC(N%d, P%d)", cm.cmId, wtrId, pageId)
	cm.ensureRecordExists(pageId)
	cm.pageRecords[pageId].recordLock.Lock(); defer cm.pageRecords[pageId].recordLock.Unlock()

	// If not processing a request, drop the WC.
	cm.requestStateLock.Lock()
	if cm.requestState != REQUEST_WQ ||  cm.requestId != reqId {
		log.Printf("CM%d: Dropped WC(N%d, P%d, %v), currently handling request %v.", cm.cmId, wtrId, pageId, reqId, cm.requestId)
		cm.requestStateLock.Unlock()
		return
	}
	cm.requestStateLock.Unlock()

	// Set new owner
	cm.pageRecords[pageId].ownerId = wtrId

	// Clear copyset
	cm.pageRecords[pageId].copySet = make([]NodeId, 0)

	// Send WC_RECV to replica, send WC_ACK to node
	cm.sendUpdateMsgToCM(reqId, MSG_CM_WC_RECV, wtrId, pageId)
	cm.send(reqId, MSG_WC_ACK, cm.cmId, wtrId, pageId)
	
	log.Printf("CM: Completed WQ(N%d, P%d, %v)", wtrId, pageId, reqId)
	cm.requestStateLock.Lock()
	cm.requestorId = InvalidNodeId
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

func (cm *CentralManager) sendEmptyUpdateMsgToCM() {
	copysets := make(map[PageId]([]NodeId))
	owners := make(map[PageId]NodeId)
	for pageId, record := range cm.pageRecords {
		copysets[pageId] = make([]NodeId, 0)
		for _, nodeId := range record.copySet {
			copysets[pageId] = append(copysets[pageId], nodeId)
		}
		owners[pageId] = record.ownerId
	}
	msg := NewCMUpdate("CM_UPDATE", MSG_CM_UPDATE, InvalidNodeId, InvalidPageId, copysets, owners)
	cm.otherInternalPort.RecvChan <- msg
}

func (cm *CentralManager) ensureRecordExists(pageId PageId) {
	if _, ok := cm.pageRecords[pageId]; !ok {
		cm.pageRecords[pageId] = &PageRecord{InvalidNodeId, make([]NodeId, 0), pageId, &sync.Mutex{}}
	}
}

func (cm *CentralManager) addToCopySet(tgtId NodeId, pageId PageId) {
	if tgtId == cm.pageRecords[pageId].ownerId {
		panic(fmt.Sprintf("CM%d: Tried to add the owner N%d to copyset of P%d.", cm.cmId, tgtId, pageId))
	}
	
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
