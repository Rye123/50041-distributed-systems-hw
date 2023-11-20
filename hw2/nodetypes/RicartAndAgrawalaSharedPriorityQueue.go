package nodetypes

import (
	"fmt"
	"log"
	"sync"
)

type RicartNode struct {
	// Basic Setup
	nodeId int
	clock ClockVal
	smPtr *SharedMemory
	endpoint RicartNodeEndpoint
	allEndpoints map[int]RicartNodeEndpoint
	nodeCount int

	// Exit Details
	exit chan bool
	exited bool

	// Variables for node's own request
	ongoingReq *sync.Mutex // Locked while the request is ONGOING, i.e. not complete
	hasOngoingReq bool
	lastReqTimestamp ClockVal // timestamp of this node's most recent request
	req_ack_lock *sync.Mutex // Lock to modify req_ack_count
	req_ack_count int // Reset to 0 when a new request starts

	// Variables for tracking requests
	queue *pqueue
}

func NewRicartNode(nodeId int, endpoints []RicartNodeEndpoint, sm *SharedMemory) *RicartNode {
	// Loop through endpoints and get self endpoint
	if len(endpoints) == 0 {
		panic("No endpoints given.")
	}

	nodeIds := make([]int, 0)
	endpointMap := make(map[int]RicartNodeEndpoint, 0)
	myEndpoint := endpoints[0]
	for _, endpoint := range(endpoints) {
		if endpoint.nodeId == nodeId {
			myEndpoint = endpoint
		}
		nodeIds = append(nodeIds, endpoint.nodeId)
		endpointMap[endpoint.nodeId] = endpoint
	}
	return &RicartNode{
		nodeId, ClockVal(0), sm, myEndpoint, endpointMap, len(endpoints),
		make(chan bool), false,
		&sync.Mutex{}, false, ClockVal(-1), &sync.Mutex{}, 0,
		newPQueue(),
	}
}

func (n *RicartNode) Init() error {
	//log.Printf("N%d: Initialised.", n.nodeId)
	go n.handleMsg()
	return nil
}

func (n *RicartNode) Shutdown() error {
	n.exit <- true
	n.exited = true
	return nil
}

// Send a message. Responsibility for updating clock is on the caller.
func (n *RicartNode) send(dstId int, action RSPQMsgAction, timestamp ClockVal) {
	// Build message
	msg := RSPQMsg{
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

func (n *RicartNode) handleMsg() {
	for {
		select {
		case rcvd_msg, ok := <-n.endpoint.recvChan:
			if !ok {
				panic(fmt.Sprintf("N%d: My channel was closed!", n.nodeId))
			}
			// Update local clock to be elementwise max + 1
			n.clock = MaxClockVal(n.clock, rcvd_msg.timestamp) + 1
			
			// We want to throw these messages to separate goroutines ASAP so we don't block the next send if any
			// WARNING: This would cause race conditions if the variables being modified don't have locks.
			switch rcvd_msg.action {
			case RSPQRequest: // REQUEST
				go n.handleRequest(rcvd_msg)
			case RSPQReqAck: // REQ_ACK
				go n.handleReqAck(rcvd_msg)
			}
		case <-n.exit:
			return
		}
	}
}

func (n *RicartNode) handleRequest(rcvd_msg RSPQMsg) {
	if !n.hasOngoingReq {
		// No ongoing requests, just ack
		n.clock++; n.send(rcvd_msg.nodeId, RSPQReqAck, n.clock)
		return
	}

	// Here, we have an ongoing request.

	// If our request is "EARLIER", add that request to the queue and return
	if n.lastReqTimestamp < rcvd_msg.timestamp {
		n.queue.Insert(rcvd_msg.nodeId, rcvd_msg.timestamp)
		return
	} else if n.lastReqTimestamp == rcvd_msg.timestamp {
		// Break tie with nodeId, LOWER ID is prioritised
		if n.nodeId < rcvd_msg.nodeId {
			n.queue.Insert(rcvd_msg.nodeId, rcvd_msg.timestamp)
			return
		}
	}

	// Otherwise, we acknowledge their request
	n.clock++
	n.send(rcvd_msg.nodeId, RSPQReqAck, n.clock)
}

func (n *RicartNode) handleReqAck(rcvd_msg RSPQMsg) {
	// Only handle REQ_ACK if we have an ongoing request
	if !n.hasOngoingReq {
		log.Printf("N%d: Received late REQ_ACK from %d", n.nodeId, rcvd_msg.nodeId)
		return
	}
	
	// Need to lock, otherwise we might have a race condition when two REQ_ACKs come in.
	n.req_ack_lock.Lock(); defer n.req_ack_lock.Unlock() 
	n.req_ack_count += 1
}



func (n *RicartNode) broadcast(action RSPQMsgAction, timestamp ClockVal) {
	for dstId := range(n.allEndpoints) {
		if dstId != n.nodeId {
			n.send(dstId, action, timestamp)
		}
	}
}	


func (n *RicartNode) AcquireLock() {
	// Block until we can obtain an ongoing request lock.
	n.ongoingReq.Lock()
	n.hasOngoingReq = true
	
	// Make a request with timestamp, and add req to queue
	n.clock++
	req_timestamp := n.clock
	n.lastReqTimestamp = req_timestamp
	n.queue.Insert(n.nodeId, req_timestamp)

	// Reset value of REQ_ACKs for current request
	n.req_ack_lock.Lock()
	n.req_ack_count = 1 // We acknowledge ourselves <3

	// BROADCAST REQUEST
	n.broadcast(RSPQRequest, req_timestamp)
	n.req_ack_lock.Unlock()

	// Block until we've received responses from all nodes
	for {
		if n.exited { return }
		n.req_ack_lock.Lock()
		if n.req_ack_count != n.nodeCount {
			n.req_ack_lock.Unlock()
			continue
		}
		n.req_ack_lock.Unlock()
		break
	}
	log.Printf("N%d: %d: Lock acquired. Entering CS. Queue: %v", n.nodeId, n.clock, n.queue.contents)

	// Enter the CS
	n.smPtr.EnterCS(n.nodeId, req_timestamp)
}

func (n *RicartNode) ReleaseLock() {
	// Exit the CS
	n.smPtr.ExitCS(n.nodeId)
	
	// Pop head of queue
	n.queue.Extract()
	log.Printf("N%d: %d: Exiting CS. Lock Released. Queue: %v", n.nodeId, n.clock, n.queue.contents)

	// Now, we can respond to all requests
	for n.queue.Length() > 0 {
		tgtId := n.queue.Extract()
		n.clock++
		resp_timestamp := n.clock
		n.send(tgtId, RSPQReqAck, resp_timestamp)
	}

	// Request is completed
	n.hasOngoingReq = false
	n.ongoingReq.Unlock()
}


type RSPQMsgAction int
const (
	RSPQRequest RSPQMsgAction = iota
	RSPQReqAck
)

type RSPQMsg struct {
	nodeId int
	timestamp ClockVal
	action RSPQMsgAction
}

type RicartNodeEndpoint struct {
	nodeId int
	recvChan chan RSPQMsg
}

func NewRicartNodeEndpoint(nodeId int) RicartNodeEndpoint {
	return RicartNodeEndpoint{
		nodeId,
		make(chan RSPQMsg),
	}
}
