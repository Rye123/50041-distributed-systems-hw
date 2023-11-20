package nodetypes

import (
	"fmt"
	"log"
	"sync"
)

type LSPQMsgAction int
const (
	LSPQRequest LSPQMsgAction = iota
	LSPQRelease
	LSPQReqAck
)

type LSPQMsg struct {
	nodeId int
	timestamp ClockVal
	action LSPQMsgAction
}

type LamportNodeEndpoint struct {
	nodeId int
	recvChan chan LSPQMsg
}

func NewLamportNodeEndpoint(nodeId int) LamportNodeEndpoint {
	return LamportNodeEndpoint{
		nodeId,
		make(chan LSPQMsg, 10000),
	}
}

type LamportNode struct {
	nodeId int
	clock ClockVal
	clockLock *sync.Mutex
	smPtr *SharedMemory
	queue *pqueue
	endpoint LamportNodeEndpoint
	allEndpoints map[int]LamportNodeEndpoint
	nodeCount int
	req_ack_lock *sync.Mutex // Lock to modify req_ack_count
	req_ack_count int // Reset to 0 when a new request starts
	lastReqTimestamp ClockVal
	ongoingReq *sync.Mutex // Locked while the request is ONGOING, i.e. not complete
	pendingReq *sync.Mutex // Locked while the request is PENDING, i.e. not fully responded to
	exit chan bool
	exited bool
}

func NewLamportNode(nodeId int, endpoints []LamportNodeEndpoint, sm *SharedMemory) *LamportNode {
	// Loop through endpoints and get self endpoint
	if len(endpoints) == 0 {
		panic("No endpoints given.")
	}

	nodeIds := make([]int, 0)
	endpointMap := make(map[int]LamportNodeEndpoint, 0)
	myEndpoint := endpoints[0]
	for _, endpoint := range(endpoints) {
		if endpoint.nodeId == nodeId {
			myEndpoint = endpoint
		}
		nodeIds = append(nodeIds, endpoint.nodeId)
		endpointMap[endpoint.nodeId] = endpoint
	}
	return &LamportNode{
		nodeId, ClockVal(0), &sync.Mutex{}, sm, newPQueue(),
		myEndpoint, endpointMap, len(endpoints),
		&sync.Mutex{}, 0, ClockVal(0),
		&sync.Mutex{}, &sync.Mutex{},
		make(chan bool), false}
}

func (n *LamportNode) Init() error {
	//log.Printf("N%d: Initialised.", n.nodeId)
	go n.handleMsg()
	return nil
}

func (n *LamportNode) Shutdown() error {
	n.exit <- true
	n.exited = true
	return nil
}

// Send a message. Responsibility for updating clock is on the caller.
func (n *LamportNode) send(dstId int, action LSPQMsgAction, timestamp ClockVal) {
	// Build message
	msg := LSPQMsg{
		n.nodeId,
		timestamp,
		action,
	}

	// Send the message
	if dstId == n.nodeId {
		panic("Attempted to send message to self")
	} else if _, ok := n.allEndpoints[dstId]; !ok {
		panic(fmt.Sprintf("N%d: Unknown endpoint with ID %d.", n.nodeId, dstId))
	}
	n.allEndpoints[dstId].recvChan <- msg
}

func (n *LamportNode) handleMsg() {
	for {
		select {
		case rcvd_msg, ok := <-n.endpoint.recvChan:
			if !ok {
				panic(fmt.Sprintf("N%d: My channel was closed!", n.nodeId))
			}
			// Update local clock to be elementwise max + 1
			n.clockLock.Lock()
			n.clock = MaxClockVal(n.clock, rcvd_msg.timestamp) + 1
			n.clockLock.Unlock()
			
			// We want to throw these messages to separate goroutines ASAP so we don't block the next send if any
			// WARNING: This would cause race conditions if the variables being modified don't have locks.
			switch rcvd_msg.action {
			case 0: // REQUEST
				go n.handleRequest(rcvd_msg)
			case 1: // RELEASE
				go n.handleRelease(rcvd_msg)
			case 2: // REQ_ACK
				go n.handleReqAck(rcvd_msg)
			}
		case <-n.exit:
			return
		}
	}
}

func (n *LamportNode) handleRequest(rcvd_msg LSPQMsg) {
	// Add request to queue
	n.queue.Insert(rcvd_msg.nodeId, rcvd_msg.timestamp)

	// Respond if we have no EARLIER outstanding requests
	// 1. Check for ongoing requests by obtaining the lock.
	if !n.ongoingReq.TryLock() {
		// Didn't get the lock -- we HAVE an ongoing request
		if n.lastReqTimestamp > rcvd_msg.timestamp {
			// OUR request is LATER, ACK the req and return
			n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
			n.send(rcvd_msg.nodeId, LSPQReqAck, n.clock)
			return
		} else if n.lastReqTimestamp == rcvd_msg.timestamp {
			// Requests are CONCURRENT
			// Break tie with nodeId, LOWER ID is prioritised
			if n.nodeId > rcvd_msg.nodeId {
				n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
				n.send(rcvd_msg.nodeId, LSPQReqAck, n.clock)
				return
			}
		}

		// Our request is EARLIER. Block until it is no longer PENDING.
		n.pendingReq.Lock()
		// Respond with REQ_ACK AFTER it's no longer PENDING
		n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
		n.send(rcvd_msg.nodeId, LSPQReqAck, n.clock)
		n.pendingReq.Unlock()
	} else {
		// We DON'T have an ongoing request at all
		// Respond with REQ_ACK
		n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
		n.send(rcvd_msg.nodeId, LSPQReqAck, n.clock)
		n.ongoingReq.Unlock()
	}
}

func (n *LamportNode) handleRelease(rcvd_msg LSPQMsg) {
	// Pop head of queue.
	n.queue.Extract()
}

func (n *LamportNode) handleReqAck(rcvd_msg LSPQMsg) {
	// Only handle REQ_ACK if we have an ongoing request
	if n.ongoingReq.TryLock() {
		log.Printf("N%d: Received late REQ_ACK from %d", n.nodeId, rcvd_msg.nodeId)
		n.ongoingReq.Unlock()
		return
	}

	n.req_ack_lock.Lock(); defer n.req_ack_lock.Unlock() // Need to lock, otherwise we might have a race condition when two REQ_ACKs come in.
	n.req_ack_count += 1

	// If request has required number, we recognise the request as no longer pending
	// The only blocker is now if the request is at the HEAD
	if n.req_ack_count == n.nodeCount {
		log.Printf("N%d: %d: Request no longer PENDING.", n.nodeId, n.clock)
		n.pendingReq.Unlock()
	} else if n.req_ack_count > n.nodeCount {
		panic(fmt.Sprintf("N%d: Has %d REQ_ACKs, expected %d", n.nodeId, n.req_ack_count, n.nodeCount))
	}
}



func (n *LamportNode) broadcast(action LSPQMsgAction, timestamp ClockVal) {
	for dstId := range(n.allEndpoints) {
		if dstId != n.nodeId {
			n.send(dstId, action, timestamp)
		}
	}
}	


func (n *LamportNode) AcquireLock() {
	// Block until we can obtain an ongoing request lock.
	n.ongoingReq.Lock()
	//log.Printf("N%d: %d: Proceeding with request.", n.nodeId, n.clock)
	
	// Make a request with timestamp, and add req to queue
	n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
	req_timestamp := n.clock
	n.lastReqTimestamp = req_timestamp
	n.queue.Insert(n.nodeId, req_timestamp)
	
	n.pendingReq.Lock()

	// Reset value of REQ_ACKs for current request
	n.req_ack_lock.Lock()
	n.req_ack_count = 1 // We acknowledge ourselves <3
	n.req_ack_lock.Unlock();

	// BROADCAST REQUEST
	n.broadcast(LSPQRequest, req_timestamp)

	// Indicate that the request is now PENDING
	//log.Printf("N%d: %d: Broadcasted request to enter", n.nodeId, req_timestamp)

	// Block until we've received responses from all nodes AND request is at head of queue
	for (n.req_ack_count < n.nodeCount || n.queue.Peek() != n.nodeId) {
		if n.exited {
			return
		}
	}
	log.Printf("N%d: %d: Lock acquired. Entering CS. Queue: %v", n.nodeId, n.clock, n.queue.contents)

	// Enter the CS
	n.smPtr.EnterCS(n.nodeId, req_timestamp)

	// Request is completed
	n.ongoingReq.Unlock()
}

func (n *LamportNode) ReleaseLock() {
	// Exit the CS
	n.smPtr.ExitCS(n.nodeId)
	
	// Pop head of queue
	n.queue.Extract()

	// Broadcast request with timestamp: RELEASE
	log.Printf("N%d: %d: Exiting CS. Lock Released. Queue: %v", n.nodeId, n.clock, n.queue.contents)
	n.clockLock.Lock(); n.clock++; n.clockLock.Unlock()
	n.broadcast(LSPQRelease, n.clock)
}
