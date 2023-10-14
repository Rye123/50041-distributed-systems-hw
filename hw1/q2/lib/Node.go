package lib

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)
type msgType int
const (
	MSG_TYPE_SYNC msgType = iota // Message sent by coordinator, containing data
	MSG_TYPE_ELECTION_START // Message sent by a node to start an election
	MSG_TYPE_ELECTION_VETO // Message sent by a higher ID node to reject an election
	MSG_TYPE_ELECTION_WIN // Message sent by a node to declare itself as coordinator
)
// If message involves an election, message will have the election ID stored in the data

type NodeId int

type Message struct {
	Type msgType
	SrcId NodeId
	DstId NodeId
	data string
}

type Node struct {
	Id NodeId
	CoordinatorId NodeId
	nodeMap map[NodeId]Node // Maps a node ID to a Node
	Data string
	SendInterval time.Duration // How often to send data
	Timeout time.Duration
	ControlChan chan Message
	DataChan chan Message
	vetoChan chan Message // Internal channel to monitor for vetoes
	ongoingElection bool // Whether or not this node has started an election
	IsAlive bool
	quitChans [](chan bool)
}

func NewNode(id NodeId, sendInterval, timeout time.Duration) *Node {
	return &Node{id, -1, make(map[NodeId]Node, 0), "", sendInterval, timeout,  make(chan Message), make(chan Message), make(chan Message), false, true, make([](chan bool), 0)}
}

func (node *Node) Initialise(nodeLs []Node) {
	for _, other := range(nodeLs) {
		if other.Id == node.Id {
			continue
		}

		node.nodeMap[other.Id] = other
	}

	node.quitChans = append(node.quitChans, make(chan bool), make(chan bool), make(chan bool))

	go node.SyncData(node.quitChans[0])
	go node.HandleControl(node.quitChans[1])
	go node.HandleData(node.quitChans[2])

	// Start election upon initialisation
	node.StartElection()
}

func (node *Node) Send(mType msgType, dstNode Node, data string) {
	if !node.IsAlive {
		return
	}

	msg := Message{mType, node.Id, dstNode.Id, data}
	log.Printf("N%d to N%d: %s", node.Id, dstNode.Id, data)
	
	dst := node.nodeMap[msg.DstId]
	if msg.Type == MSG_TYPE_SYNC {
		dst.DataChan <- msg
	} else {
		dst.ControlChan <- msg
	}
}

func (node *Node) SyncData(quit chan bool) {
	for {
		select {
		case <-time.After(node.SendInterval):
			if node.Id != node.CoordinatorId {
				continue
			}
			for nodeId := range(node.nodeMap) {
				node.Send(MSG_TYPE_SYNC, node.nodeMap[nodeId], node.Data)
			}
			
			
		}
	}
}

func (node *Node) HandleControl(quit chan bool) {
	for {
		if !node.IsAlive {
			continue
		}
		
		// Control message: Either an ELECTION_START, ELECTION_VETO or ELECTION_WIN		
		msg, ok := <-node.ControlChan

		if !ok {
			log.Printf("Error: Closure of control channel by node %d", node.Id)
		}

		switch msg.Type {
		case MSG_TYPE_ELECTION_VETO:
			// If veto, then it's handled by the goroutines in StartElection.
			log.Printf("N%d: Received ELECTION_VETO from %d.", node.Id, msg.SrcId)
			node.vetoChan <- msg
		case MSG_TYPE_ELECTION_START:
			// If another node is starting an election, REJECT if id lower
			// Note that we use the same data in the message (i.e. the election ID)
			log.Printf("N%d: Received ELECTION_START from %d.", node.Id, msg.SrcId)
			if msg.SrcId < node.Id {
				node.Send(MSG_TYPE_ELECTION_VETO, node.nodeMap[msg.SrcId], msg.data)
			}
		case MSG_TYPE_ELECTION_WIN:
			// If another node declares it has won, start an election if lower
			// Otherwise accept
			log.Printf("N%d: Received ELECTION_WIN from %d.", node.Id, msg.SrcId)
			if msg.SrcId < node.Id {
				node.StartElection()
			} else {
				node.CoordinatorId = msg.SrcId
			}
		}
	}
	
}

func (node *Node) HandleData(quit chan bool) {
	/**
	  We expect a data message at least every (node.SendInterval + node.Timeout/2) seconds.
	  - node.Timeout is the simulated time taken to get a response after sending a request (i.e. RTT)
	  - Hence, RTT/2 gives the amount of time a message SHOULD take to come from the server
	*/
	for {
		if !node.IsAlive {
			continue
		}
		
		select {
		case msg, ok := <-node.DataChan:
			// Data message
			if !ok {
				log.Printf("Error: Closure of data channel by node %d", node.Id)
			}
			if msg.SrcId > node.Id {
				// i don't take orders from you!
				node.StartElection()
				continue
			}
			node.Data = msg.data
		case <-time.After(node.SendInterval + node.Timeout):
			if node.Id == node.CoordinatorId {
				// I'm the coordinator, why would i time out???
				continue
			}
			// Assume coordinator is down
			log.Printf("N%d: Coordinator is down.", node.Id)
			node.StartElection()
		}
	}
}

func (node *Node) UpdateData(data string) {
	if node.Id != node.CoordinatorId {
		return
	}

	node.Data = data
}

func (node *Node) StartElection() {
	if node.ongoingElection {
		log.Printf("N%d: Cannot start election, ongoing one.", node.Id)
		return
	}
	log.Printf("N%d: Starting election.", node.Id)
	node.ongoingElection = true
	
	// Generate random election ID
	elId := fmt.Sprintf("EL%d", rand.Int())
	
	// Start goroutines asking higher IDs
	completedChan := make(chan bool)
	stopReqChans := make([](chan bool), 0)
	for nodeId := range(node.nodeMap) {
		if nodeId <= node.Id {
			continue
		}

		nodeId := nodeId
		stopReqChan := make(chan bool)
		stopReqChans = append(stopReqChans, stopReqChan)
		go func(dst Node, stop chan bool) {
			// Send the request
			node.Send(MSG_TYPE_ELECTION_START, dst, elId)

			// Watch for either: timeout, or stopReqChan
			select {
			case <-stop: // signalled to stop by a veto
				break
			case <- time.After(node.Timeout):
				break
			}
			completedChan <- true
			
		}(node.nodeMap[nodeId], stopReqChan)
	}

	// If we receive any vetoes MATCHING THE ELECTION ID, call all existing goroutines to stop
	veto := false
	completed := 0
	for completed < len(stopReqChans) {
		select {
		case msg := <-node.vetoChan:
			// If not match, veto must've been from an older election
			// This is because we only have one election at a time
			if msg.data != elId {
				continue
			}

			// Otherwise, it's a veto, and we stop all the others
			veto = true

			for _, stopReqChan := range(stopReqChans) {
				stopReqChan <- true
			}
			
		case <-completedChan: // we expect a total of len(stopReqChans)
			completed += 1
		}
	}

	// Announcement Stage
	log.Printf("N%d: Reached announcement stage.", node.Id)
	if veto {
		node.ongoingElection = false
		return
	}

	node.CoordinatorId = node.Id

	for nodeId := range(node.nodeMap) {
		if nodeId == node.Id {
			continue
		}
		node.Send(MSG_TYPE_ELECTION_WIN, node.nodeMap[nodeId], elId)
	}

	node.ongoingElection = false
}

// Simulate this node going down.
// To simulate a node going down in a network, we simply stop
// sending messages.
func (node *Node) Kill() {
	node.IsAlive = false
}

func (node *Node) Restart() {
	node.IsAlive = true
	node.StartElection()
}

// Actual teardown of this node.
func (node *Node) Exit() {
	
}
