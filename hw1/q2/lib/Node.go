package lib

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type msgType string

const (
	MSG_TYPE_SYNC           msgType = "SYNC"           // Sent by coordinator, containing data
	MSG_TYPE_ELECTION_START         = "ELECTION_START" // Sent by a node to start an election
	MSG_TYPE_ELECTION_VETO          = "ELECTION_VETO"  // Sent by a higher ID node to reject an election
	MSG_TYPE_ELECTION_WIN           = "ELECTION_WIN"   // Sent by a node to declare self as coordinator
)

// A standard message sent between nodes.
// The `Data` field contains either the data to be exchanged in a `MSG_TYPE_SYNC` message,
// or the election ID if the message type is about an election
type Message struct {
	Type  msgType
	SrcId NodeId
	DstId NodeId
	Data  string
}

type NodeId int

// Endpoints for a node
// This is abstracted from the node itself, to ensure we don't accidentally
// share other data between goroutines. This is the only thing that
// other nodes have access to -- forcing us to pass all data through these
// channels.
type NodeEndpoint struct {
	Id          NodeId
	ControlChan chan Message // Channel for control messages
	DataChan    chan Message // Channel for actual data
}

// A node in the system.
type Node struct {
	Id, CoordinatorId NodeId
	Data              string                  // The data structure to be synchronised.
	IsAlive           bool                    // Simulated liveness to simulate a fault.
	Endpoint          NodeEndpoint            // This node's endpoint
	endpoints         map[NodeId]NodeEndpoint // Maps a node ID to their endpoint
	sendIntv          time.Duration           // How often data is to be sent from the coordinator
	timeout           time.Duration           // Estimated RTT for messages
	vetoChan          chan Message            // Internal channel to monitor for vetoes during an election
	ongoingElectionId string                  // Election ID of current election ("" if no election)
	quitChan          chan bool               // Internal channels to kill goroutines
	disableElection   bool                    // Internal control to disable this ndoe starting elections
	electionLock      *sync.Mutex
}

// Creates a new node.
func NewNode(id NodeId, sendInterval, timeout time.Duration, disableElection bool) *Node {
	return &Node{
		id, -1, "", true,
		NodeEndpoint{id, make(chan Message), make(chan Message)},
		make(map[NodeId]NodeEndpoint, 0),
		sendInterval, timeout,
		make(chan Message, 100),
		"", make(chan bool),
		disableElection,
		&sync.Mutex{},
	}
}

// Given a list of endpoints of nodes, initialise the Node.
func (node *Node) Initialise(endpoints []NodeEndpoint) {
	for _, other := range endpoints {
		if other.Id == node.Id {
			continue
		}
		node.endpoints[other.Id] = other
	}

	// Start up goroutines to handle necessary incoming messages
	go node.HandleControl()
	go node.HandleData()
	go node.SyncData()

	node.StartElection() // Start election upon initialisation
}

// Coordinator function to send data, if this node is the coordinator.
func (node *Node) SyncData() {
	for {
		select {
		case <-time.After(node.sendIntv):
			// Don't send if you're not the coordinator
			if node.Id != node.CoordinatorId {
				continue
			}

			// Don't broadcast if you're dead
			if !node.IsAlive {
				continue
			}

			// Broadcast this node's data
			for nodeId := range node.endpoints {
				node.send(MSG_TYPE_SYNC, node.endpoints[nodeId], node.Data)
			}
		case <-node.quitChan:
			log.Printf("N%d: Shutting down SyncData.", node.Id)
			return
		}
	}
}

// Handle control messages
func (node *Node) HandleControl() {
	for {
		select {
		case msg, ok := <-node.Endpoint.ControlChan:
			if !ok {
				log.Printf("N%d: ControlChan closed, shutting down HandleControl", node.Id)
				return
			}

			if msg.SrcId == node.Id {
				panic(fmt.Sprintf("N%d: Received a message from itself", node.Id))
			}

			if msg.DstId != node.Id {
				panic(fmt.Sprintf("N%d: Received message bound for N%d.", node.Id, msg.DstId))
			}

			// Drop message if dead
			if !node.IsAlive {
				log.Printf("N%d: Dropped incoming message from %d, node is dead.", node.Id, msg.SrcId)
				continue
			}

			// Handle message
			switch msg.Type {
			case MSG_TYPE_SYNC:
				panic(fmt.Sprintf("N%d: Received MSG_TYPE_SYNC on ControlChan", node.Id))
			case MSG_TYPE_ELECTION_START:
				// If another node is starting an election, reject if ID lower and start an election
				// log.Printf("N%d: Received ELECTION_START from N%d.", node.Id, msg.SrcId)
				if msg.SrcId < node.Id {
					node.send(
						MSG_TYPE_ELECTION_VETO,
						node.endpoints[msg.SrcId],
						msg.Data, // Note we use the same election ID.
					)
					node.StartElection()
				}
			case MSG_TYPE_ELECTION_VETO:
				// If we receive a veto:
				// pass it along to the StartElection if we have an ongoing election
				if msg.Data == node.ongoingElectionId {
					log.Printf("N%d: Received ELECTION_VETO from N%d.", node.Id, msg.SrcId)
					go func() { node.vetoChan <- msg }() // Just throw it, it will be received
				}
			case MSG_TYPE_ELECTION_WIN:
				// If another node declares it has won...
				log.Printf("N%d: Received ELECTION_WIN from N%d.", node.Id, msg.SrcId)
				if msg.SrcId < node.Id {
					// (politely) remind everyone who the boss is
					node.StartElection()
				} else {
					// ok you win
					node.CoordinatorId = msg.SrcId
				}
			}
		case <-node.quitChan:
			log.Printf("N%d: Shutting down HandleControl.", node.Id)
			return
		}
	}
}

// Handle data messages
func (node *Node) HandleData() {
	// We expect a data message AT LEAST every (node.sendIntv + node.timeout/2) seconds.
	// node.sendIntv is the interval the coordinator should send a message
	// node.timeout is the (simulated) time taken to get a response after sending a request, i.e. RTT
	// Hence, RTT/2 gives the expected time taken for a msg to reach this node from the coordinator
	// Hence, (node.sendIntv + RTT/2) is the MAX time taken for a msg from the coordinator to come.

	for {
		select {
		case msg, ok := <-node.Endpoint.DataChan:
			if !ok {
				log.Printf("N%d: DataChan closed, shutting down HandleData", node.Id)
				return
			}

			// Drop message if dead
			if !node.IsAlive {
				continue
			}

			if msg.SrcId == node.Id {
				panic(fmt.Sprintf("N%d: Received a message from itself", node.Id))
			} else if msg.SrcId < node.Id {
				// I don't take orders from you!!!
				node.StartElection()
				continue
			}

			log.Printf("N%d: Received SYNC from N%d: %v", node.Id, msg.SrcId, msg.Data)
			node.Data = msg.Data
		case <-time.After(node.sendIntv + (node.timeout / 2)):
			if node.disableElection || node.Id == node.CoordinatorId || !node.IsAlive {
				continue
			}

			// Assume coordinator is down.
			log.Printf("N%d: Detected coordinator is down.", node.Id)
			node.StartElection()
		case <-node.quitChan:
			log.Printf("N%d: Shutting down HandleData", node.Id)
			return
		}
	}
}

// Initiate an election
func (node *Node) StartElection() {
	if node.disableElection {
		return
	}

	// Prevent a node from starting an election if it has already started one.
	if node.ongoingElectionId != "" {
		return
	}

	// We use TryLock here because we really don't have to re-acquire the lock to start an election if we already started an election
	if !node.electionLock.TryLock() {
		return
	}

	// Acquired the lock, start a goroutine to manage the election while we continue on
	go func() {
		node.ongoingElectionId = fmt.Sprintf("EL%d", rand.Int()) // Random election ID
		log.Printf("N%d: Starting election with ID: %v", node.Id, node.ongoingElectionId)

		defer func() {
			// Election is over, clear election veto channel
			for len(node.vetoChan) > 0 {
				<-node.vetoChan
			}
			node.ongoingElectionId = ""
			node.electionLock.Unlock()
		}()

		veto := false

		for nodeId := range node.endpoints {
			nodeId := nodeId
			if nodeId <= node.Id {
				continue
			}
			go node.send(MSG_TYPE_ELECTION_START, node.endpoints[nodeId], node.ongoingElectionId)
		}

		// Watch for timeout or vetoes
		select {
		case <-node.vetoChan:
			// Veto from ANY higher node considered as veto
			veto = true
		case <-time.After(node.timeout):
			// No responses received from any node
		}

		// Announcement Stage
		if veto {
			log.Printf("N%d: Lost election %s.", node.Id, node.ongoingElectionId)
		} else {
			log.Printf("N%d: Won election %s.", node.Id, node.ongoingElectionId)

			node.CoordinatorId = node.Id
			for nodeId := range node.endpoints {
				if nodeId >= node.Id {
					continue
				}
				node.send(
					MSG_TYPE_ELECTION_WIN,
					node.endpoints[nodeId],
					node.ongoingElectionId,
				)
			}
		}
	}()
}

// Manual update of data
func (node *Node) PushUpdate(data string) {
	if node.Id != node.CoordinatorId {
		panic(fmt.Sprintf("PushUpdate error: N%d is not the coordinator.", node.Id))
	}

	node.Data = data
}

func (node *Node) send(mType msgType, dstEndpoint NodeEndpoint, data string) {
	if !node.IsAlive {
		return
	}

	if dstEndpoint.Id == node.Id {
		panic(fmt.Sprintf("N%d: Tried to send data to itself: %s", node.Id, data))
	}

	msg := Message{mType, node.Id, dstEndpoint.Id, data}
	//log.Printf("N%d: Sent %s to N%d: %s", msg.SrcId, mType, msg.DstId, data)

	if mType == MSG_TYPE_SYNC {
		go func() { dstEndpoint.DataChan <- msg }()
	} else {
		go func() { dstEndpoint.ControlChan <- msg }()
	}
}

// Simulate this node going down.
// To simulate a node going down in a network, we simply stop
// sending messages.
func (node *Node) Kill() {
	log.Printf("N%d: Killed.", node.Id)
	node.CoordinatorId = -1
	node.IsAlive = false
}

func (node *Node) Restart() {
	node.IsAlive = true
	go node.StartElection()
}

// Actual teardown of this node.
func (node *Node) Exit() {
	node.disableElection = true // Disable just to prevent additional messages

	// Since the above goroutines don't check for quit if they're 'not alive', make them alive
	node.IsAlive = true

	// Kill all existing goroutines
	for i := 0; i < 3; i++ {
		node.quitChan <- true
	}
}
