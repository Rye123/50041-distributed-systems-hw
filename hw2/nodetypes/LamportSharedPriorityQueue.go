package nodetypes

import (
	"1005129_RYAN_TOH/hw2/clock"
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
	timestamp clock.ClockVal
	action LSPQMsgAction
}

type LamportNodeEndpoint struct {
	nodeId int
	recvChan chan LSPQMsg
}

func NewLamportNodeEndpoint(nodeId int) LamportNodeEndpoint {
	return LamportNodeEndpoint{
		nodeId,
		make(chan LSPQMsg),
	}
}

type LamportNode struct {
	nodeId int
	clock clock.ClockVal
	smPtr *SharedMemory
	queue *pqueue
	endpoint LamportNodeEndpoint
	allEndpoints map[int]LamportNodeEndpoint
	nodeCount int
	req_ack_count int // Reset to 0 when a new request starts
	ongoingReq *sync.Mutex // This is locked if we have an ongoing request
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
	return &LamportNode{nodeId, clock.NewClockVal(nodeIds), sm, newPQueue(), myEndpoint, endpointMap, len(endpoints), 0, &sync.Mutex{}}
}

func (n *LamportNode) Init() {
	log.Printf("N%d: Initialised.", n.nodeId)
	go n.handleMsg()
}

// Send a message. Responsibility for updating clock is on the caller.
func (n *LamportNode) send(dstId int, action LSPQMsgAction, timestamp clock.ClockVal) {
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

	// Log
	msgType := ""
	switch action {
	case LSPQRequest:
		msgType = "REQUEST"
	case LSPQRelease:
		msgType = "RELEASE"
	case LSPQReqAck:
		msgType = "REQ_ACK"
	default:
		panic(fmt.Sprintf("N%d: Unknown message type %d", n.nodeId, action))
		
	}
	log.Printf("N%d -> N%d: Sent %v (Timestamp: %v)", n.nodeId, dstId, msgType, timestamp)
}

func (n *LamportNode) handleMsg() {
	for {
		rcvd_msg, ok := <-n.endpoint.recvChan
		if !ok {
			panic(fmt.Sprintf("N%d: My channel was closed!", n.nodeId))
		}

		// Update clock, check for causality violation
		// 1. If local clock > received clock, potential causality violation
		if n.clock.Compare(rcvd_msg.timestamp) == 1 {
			log.Printf("N%d: Detected potential causality violation with message from N%d. \n\tLocal clock: %v; Received Clock: %v", n.nodeId, rcvd_msg.nodeId, n.clock, rcvd_msg.timestamp)
			// TODO: rejection msg?
			continue
		}
		// 2. Update local clock to be elementwise max + 1
		n.clock = clock.MaxClockValue(n.clock, rcvd_msg.timestamp).Increment(n.nodeId, 1)


		// Log
		msgType := ""
		switch rcvd_msg.action {
		case 0:
			msgType = "REQUEST"
		case 1:
			msgType = "RELEASE"
		case 2:
			msgType = "REQ_ACK"
		default:
			panic(fmt.Sprintf("N%d: Unknown message type %d", n.nodeId, rcvd_msg.action))
		}
		log.Printf("N%d: Received %v from N%d.", n.nodeId, msgType, rcvd_msg.nodeId)
		
		// We want to throw these messages to separate goroutines ASAP so we don't block the next send if any
		switch rcvd_msg.action {
		case 0: // REQUEST
			go n.handleRequest(rcvd_msg)
		case 1: // RELEASE
			go n.handleRelease(rcvd_msg)
		case 2: // REQ_ACK
			go n.handleReqAck(rcvd_msg)
		}
	}
}

func (n *LamportNode) handleRequest(rcvd_msg LSPQMsg) {
	// Add request to queue
	n.queue.Insert(rcvd_msg.nodeId, rcvd_msg.timestamp.Clone())

	// Respond if we have no outstanding requests
	// 1. Here, since ongoingReq is LOCKED if we have an ongoing request, obtaining the lock suggests that there's no ongoing request.
	n.ongoingReq.Lock() // Block until we get it
	// 2. Respond with req_ack
	n.clock = n.clock.Increment(n.nodeId, 1)
	n.send(rcvd_msg.nodeId, LSPQReqAck, n.clock.Clone())
	// 3. Release ongoingReq lock
	n.ongoingReq.Unlock()
	
}

func (n *LamportNode) handleRelease(rcvd_msg LSPQMsg) {
	// Pop head of queue.
	n.queue.Extract()
}

func (n *LamportNode) handleReqAck(rcvd_msg LSPQMsg) {
	// add to req_ack_chan only if we have an ongoing request
	if n.ongoingReq.TryLock() {
		n.ongoingReq.Unlock()
		return
	}
	
	// TODO: add ID for each request
	n.req_ack_count += 1
}



func (n *LamportNode) broadcast(action LSPQMsgAction, timestamp clock.ClockVal) {
	for dstId := range(n.allEndpoints) {
		if dstId != n.nodeId {
			n.send(dstId, action, timestamp)
		}
	}
}	


func (n *LamportNode) AcquireLock() {
	if !n.ongoingReq.TryLock() {
		// If we have an ongoing request, then ignore this request.
		// TODO: do we need to use a queue for this
		log.Printf("N%d: Couldn't acquire lock, already have an ongoing request.", n.nodeId)
		return
	}
	defer n.ongoingReq.Unlock()
	
	// Make a request with timestamp, and add req to queue
	n.clock = n.clock.Increment(n.nodeId, 1)
	req_timestamp := n.clock.Clone()
	n.queue.Insert(n.nodeId, req_timestamp)
	n.req_ack_count = 1 // We acknowledge ourselves <3
	n.broadcast(LSPQRequest, req_timestamp) // BROADCAST REQUEST

	// Block until we've received responses from all nodes AND request is at head of queue
	for (n.req_ack_count < n.nodeCount || n.queue.Peek() != n.nodeId) {}

	log.Printf("N%d: Entering CS.", n.nodeId)

	// Enter the CS
	n.smPtr.EnterCS(n.nodeId, n.clock.Clone())
}

func (n *LamportNode) ReleaseLock() {
	// Exit the CS
	n.smPtr.ExitCS(n.nodeId)
	
	// Pop head of queue
	n.queue.Extract()

	// Broadcast request with timestamp: RELEASE
	n.clock = n.clock.Increment(n.nodeId, 1)
	n.broadcast(LSPQRelease, n.clock.Clone())
}
