package nodetypes

import (
	"fmt"
	"log"
	"sync"
)

// Communication endpoint for each node -- this is the only thing shared between nodes, which allows us to prevent nodes from modifying each others' memory
type VoterNodeEndpoint struct {
	nodeId int
	recvChan chan VoterMsg // Incoming messages for this node are sent here
}

func NewVoterNodeEndpoint(nodeId int) VoterNodeEndpoint {
	return VoterNodeEndpoint{nodeId, make(chan VoterMsg)}
}

// Node that follows the Voting Protocol.
type VoterNode struct {
	// Basic Setup
	nodeId int
	clock ClockVal
	smPtr *SharedMemory
	endpoint VoterNodeEndpoint
	allEndpoints map[int]VoterNodeEndpoint
	nodeCount int

	// Exit details
	exit chan bool
	voterExit chan bool
	reqSeshExit chan bool

	// Variables for node's own request
	ongoingReqLock *sync.Mutex // Locked while the node is REQUESTING
	hasOngoingReq bool
	ongoingReqSesh *voteSession
	electionCount int // Number of elections so far

	// Variables to handle node's vote
	voteBacklog *pqueueVP
	attemptingRescind bool     // True if this node has rescinded but is waiting for the vote
	votedForReq VoterMsg // Request that we've voted for
	voteMsgs chan VoterMsg // Channel of unhandled requests/releases (i.e. not in queue)
}

// Initialise a new VoterNode. This connects the node to SharedMemory, and to the other nodes' endpoints.
func NewVoterNode(nodeId int, endpoints []VoterNodeEndpoint, sm *SharedMemory) *VoterNode {
	if len(endpoints) == 0 {
		panic("No endpoints given.")
	}

	nodeIds := make([]int, 0)
	endpointMap := make(map[int]VoterNodeEndpoint, 0)
	myEndpoint := endpoints[0]
	for _, endpoint := range(endpoints) {
		if endpoint.nodeId == nodeId {
			myEndpoint = endpoint
		}
		nodeIds = append(nodeIds, endpoint.nodeId)
		endpointMap[endpoint.nodeId] = endpoint
	}
	return &VoterNode{
		nodeId, ClockVal(0), sm, myEndpoint, endpointMap, len(endpoints),
		make(chan bool), make(chan bool), make(chan bool),
		&sync.Mutex{}, false, newVoteSession("", ClockVal(-1), endpointMap, 0, make(chan bool)), 0,
		newPQueueVP(), false, getEmptyVoterMsg(), make(chan VoterMsg),
	}
}

func (n *VoterNode) Init() error {
	go n.handleMsg(); go n.Voter()
	return nil
}

func (n *VoterNode) Shutdown() error {
	n.exit <- true; n.voterExit <- true;
	if n.hasOngoingReq {
		n.reqSeshExit <- true
	}
	return nil
}

// Handle an incoming message
func (n *VoterNode) handleMsg() {
	for {
		select {
		case rcvd_msg, ok := <-n.endpoint.recvChan:
			if !ok {
				panic(fmt.Sprintf("N%d: My channel was closed!", n.nodeId))
			}
			// Update local clock to be elementwise max + 1
			n.clock = MaxClockVal(n.clock, rcvd_msg.timestamp) + 1
			
			// We want to throw these messages to separate goroutines ASAP so we don't block the next send if any
			switch rcvd_msg.action {
			case VPRequest:
				// Handover to the Voter goroutine
				go func(msg VoterMsg){n.voteMsgs <- msg}(rcvd_msg)
			case VPVote:
				go n.handleVote(rcvd_msg)
			case VPRescind:
				go n.handleRescind(rcvd_msg)
			case VPRelease:
				// Handover to the Voter goroutine
				go func(msg VoterMsg){n.voteMsgs <- msg}(rcvd_msg)
			}
			//log.Printf("[%d] - N%d: Received %v from N%d.", n.clock, n.nodeId, getVoterMsgAction(rcvd_msg.action), rcvd_msg.nodeId)
		case <-n.exit:
			return
		}
	}
}

// Manages this node's vote -- ensures synchronous management of the vote
func (n *VoterNode) Voter() {
	for {
		select {
		case msg := <-n.voteMsgs:
			switch msg.action {
			case VPRequest:
				n.voterHandleRequest(msg)
			case VPRelease:
				n.voterHandleRelease(msg)
			}
		case <-n.voterExit:
			for range n.voteMsgs {
				// drop messages
			}
			close(n.voteMsgs)
			return
		}
		
	}
}

// Handle an incoming request.
func (n *VoterNode) voterHandleRequest(msg VoterMsg) {
	if n.votedForReq.action == VPInvalid {
		// We haven't voted!
		n.votedForReq = msg
		n.clock++; n.send(msg.electionId, msg.nodeId, VPVote, n.clock)
		log.Printf("[%d] - N%d: VOTED for N%d.", n.clock, n.nodeId, msg.nodeId)
		return
	}

	// We've voted already -- either we rescind, or add to backlog
	rescind := false
	if msg.timestamp < n.votedForReq.timestamp {
		rescind = true
	} else if msg.timestamp == n.votedForReq.timestamp {
		if msg.nodeId < n.votedForReq.nodeId {
			// Prioritise lower node ID
			rescind = true
		}
	}

	if rescind {
		// Insert older req in case we cannot rescind
		n.voteBacklog.Insert(msg.nodeId, msg.electionId, msg.timestamp)

		// Request for rescind
		if !n.attemptingRescind {
			// We only send the request to rescind if we haven't sent it before
			laterReq := n.votedForReq
			n.clock++; n.send(laterReq.electionId, laterReq.nodeId, VPRescind, n.clock)
			log.Printf("[%d] - N%d: ATTEMPT RESCIND from N%d.", n.clock, n.nodeId, laterReq.nodeId)
			n.attemptingRescind = true
		}
	} else {
		// Add to backlog
		n.voteBacklog.Insert(msg.nodeId, msg.electionId, msg.timestamp)
	}
}

// Handle an incoming release
func (n *VoterNode) voterHandleRelease(msg VoterMsg) {
	if msg.electionId != n.votedForReq.electionId {
		panic(fmt.Sprintf("N%d: Received INVALID RELEASE from N%d: Expected ID: %v, received ID: %v", n.nodeId, msg.nodeId, n.votedForReq.electionId, msg.electionId))
		
	}

	n.attemptingRescind = false
	
	if n.voteBacklog.Length() == 0 {
		n.votedForReq = getEmptyVoterMsg()
		log.Printf("[%d] - N%d: Received RELEASE from N%d. Current backlog: []", n.clock, n.nodeId, msg.nodeId)
		return
	}

	// Vote for next in queue
	next := n.voteBacklog.ExtractElem()
	n.votedForReq = VoterMsg{next.nodeId, next.electionId, next.timestamp, VPRequest}
	n.clock++; n.send(next.electionId, next.nodeId, VPVote, n.clock)
	
	log.Printf("[%d] - N%d: Received RELEASE from N%d. VOTED for N%d.", n.clock, n.nodeId, msg.nodeId, next.nodeId)
}

func (n *VoterNode) handleVote(rcvd_msg VoterMsg) {
	if !n.hasOngoingReq {
		log.Printf("[%d] - N%d: Received late VOTE from N%d.", n.clock, n.nodeId, rcvd_msg.nodeId)
		
		// Return the late vote
		n.clock++; n.send(rcvd_msg.electionId, rcvd_msg.nodeId, VPRelease, n.clock)
		return
	}

	successfulVote := n.ongoingReqSesh.AddVote(rcvd_msg.electionId, rcvd_msg.nodeId)

	if successfulVote {
		log.Printf("[%d] - N%d: Received VOTE from N%d. Voters: %v", n.clock, n.nodeId, rcvd_msg.nodeId, n.ongoingReqSesh.GetVoters())
	} else {
		log.Printf("[%d] - N%d: Received late VOTE from N%d.", n.clock, n.nodeId, rcvd_msg.nodeId)
		
		// Return the late vote
		n.clock++; n.send(rcvd_msg.electionId, rcvd_msg.nodeId, VPRelease, n.clock)
	}
}

// Handle an incoming rescind
func (n *VoterNode) handleRescind(rcvd_msg VoterMsg) {
	if !n.hasOngoingReq {
		log.Printf("[%d] - N%d: Received late RESCIND from N%d.", n.clock, n.nodeId, rcvd_msg.nodeId)
		return
	}

	successfulRescind := n.ongoingReqSesh.RemoveVote(rcvd_msg.electionId, rcvd_msg.nodeId)
	if successfulRescind {
		// Send release
		n.clock++; n.send(rcvd_msg.electionId, rcvd_msg.nodeId, VPRelease, n.clock)

		// Re-send request
		n.send(n.ongoingReqSesh.electionId, rcvd_msg.nodeId, VPRequest, n.ongoingReqSesh.req_ts)

		log.Printf("[%d] - N%d: Received RESCIND from N%d. Released N%d's vote.", n.clock, n.nodeId, rcvd_msg.nodeId, rcvd_msg.nodeId)
	} else {
		log.Printf("[%d] - N%d: Received late RESCIND from N%d.", n.clock, n.nodeId, rcvd_msg.nodeId)
	}
}

// Attempt to acquire the lock
func (n *VoterNode) AcquireLock() {
	// Block until we can start an ongoing request
	n.ongoingReqLock.Lock(); n.hasOngoingReq = true
	n.clock++; req_ts := n.clock

	// Initialise vote tracking
	electionId := fmt.Sprintf("N%d-TS%d-EL%d", n.nodeId, n.clock, n.electionCount)
	n.electionCount++
	n.ongoingReqSesh = newVoteSession(electionId, req_ts, n.allEndpoints, (n.nodeCount/2)+1, n.reqSeshExit)

	// Make request
	n.broadcast(electionId, VPRequest, req_ts)
	log.Printf("[%d] - N%d: Start request", n.clock, n.nodeId)

	// BLOCK until we have majority responses
	successfulVote := n.ongoingReqSesh.Wait()
	if !successfulVote { return } // node was shut down

	log.Printf("[%d] - N%d: Lock acquired. Entering CS. Voters: %v", n.clock, n.nodeId, n.ongoingReqSesh.GetVoters())

	n.smPtr.EnterCS(n.nodeId, req_ts) // (Simulated method of entering unsafe CS)
}

// Attempt to release the lock
func (n *VoterNode) ReleaseLock() {
	n.smPtr.ExitCS(n.nodeId)  // (Simulated method of exiting unsafe CS)

	// Send releases
	voters := n.ongoingReqSesh.GetVoters()
	n.clock++; release_ts := n.clock
	log.Printf("[%d] - N%d: Lock released", n.clock, n.nodeId)
	
	for _, voterId := range(voters) {
		log.Printf("[%d] - N%d: Sent RELEASE to N%d.", n.clock, n.nodeId, voterId)
		n.send(n.ongoingReqSesh.electionId, voterId, VPRelease, release_ts)
	}
	n.ongoingReqLock.Unlock(); n.hasOngoingReq = false
}


// Send a message. Responsibility for updating clock is on the caller.
func (n *VoterNode) send(elId string, dstId int, action VoterMsgAction, timestamp ClockVal) {
	// Build message
	msg := VoterMsg{
		n.nodeId,
		elId,
		timestamp,
		action,
	}

	// Send the message
	if _, ok := n.allEndpoints[dstId]; !ok {
		panic(fmt.Sprintf("N%d: Tried to send %v to unknown endpoint with ID %d.", n.nodeId, getVoterMsgAction(action), dstId))
	}
	n.allEndpoints[dstId].recvChan <- msg
}

// Broadcast to ALL including self
func (n *VoterNode) broadcast(elId string, action VoterMsgAction, timestamp ClockVal) {
	for dstId := range(n.allEndpoints) {
		n.send(elId, dstId, action, timestamp)
	}
}

// -----
// HELPER STRUCTS
// -----

// --- VOTERMSG ---
// Message used for inter-VoterNode communication.
type VoterMsg struct {
	nodeId int
	electionId string
	timestamp ClockVal
	action VoterMsgAction
}

func getEmptyVoterMsg() VoterMsg {
	return VoterMsg{-1, "", ClockVal(-1), VPInvalid}
}

type VoterMsgAction int
const (
	VPRequest VoterMsgAction = iota
	VPVote
	VPRescind
	VPRelease
	VPInvalid
)

func getVoterMsgAction(action VoterMsgAction) string {
	switch action {
	case VPRequest:
		return "REQUEST"
	case VPVote:
		return "VOTE"
	case VPRescind:
		return "RESCIND"
	case VPRelease:
		return "RELEASE"
	case VPInvalid:
		return "INVALID"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", action)
	}
}


// Internal vote session tracker -- this allows us to receive RESCIND messages and still receive the delayed VOTE
type voteSession struct {
	electionId string
	req_ts ClockVal
	votes map[int]int // links nodes to the status of their votes -- 1 for VOTED
	sessionLock *sync.Mutex
	doneChan chan bool
	exitChan chan bool
	majority int
}

func newVoteSession(electionId string, req_ts ClockVal, endpointMap map[int]VoterNodeEndpoint, expectedMajority int, exitChan chan bool) *voteSession {
	voteMap := make(map[int]int)
	for nodeId := range(endpointMap) {
		voteMap[nodeId] = 0
	}
	
	return &voteSession{electionId, req_ts, voteMap, &sync.Mutex{}, make(chan bool), exitChan, expectedMajority}
}

func (s *voteSession) AddVote(electionId string, nodeId int) bool {
	s.sessionLock.Lock(); defer s.sessionLock.Unlock()
	if electionId != s.electionId {
		return false
	}

	if s.checkVotes() == s.majority { // we REJECT any further votes
		// This prevents the case where this node receives a new vote while it is releasing the votes and forgets the received node.
		return false
	}

	if s.votes[nodeId] == -1 { // RESCINDED
		s.votes[nodeId] = 0
	} else if s.votes[nodeId] == 0 { // NOT VOTED
		s.votes[nodeId] = 1
	} else {
		panic(fmt.Sprintf("s.votes[%d] = %d", nodeId, s.votes[nodeId] + 1))
	}

	if s.checkVotes() == s.majority {
		s.doneChan <- true
	}

	return true
}

func (s *voteSession) RemoveVote(electionId string, nodeId int) bool {
	s.sessionLock.Lock(); defer s.sessionLock.Unlock()
	if electionId != s.electionId {
		return false
	}
	
	if s.checkVotes() >= s.majority { // too late, prevent any further modification
		return false
	}

	if s.votes[nodeId] == -1 { // RESCINDED
		panic(fmt.Sprintf("s.votes[%d] = %d", nodeId, s.votes[nodeId] - 1))
	} else if s.votes[nodeId] == 0 { // NOT VOTED
		s.votes[nodeId] = -1
	} else {
		s.votes[nodeId] = 0
	}

	return true
}

func (s *voteSession) checkVotes() int {
	// Loop through, ignoring votes that are -1 and 0:
	votes := 0
	for _, value := range s.votes {
		if value == 1 {
			votes++
		}
	}
	return votes
}

func (s *voteSession) GetVoters() []int {
	s.sessionLock.Lock(); defer s.sessionLock.Unlock()
	ret := make([]int, 0)
	for voterId, value := range s.votes {
		if value == 1 {
			ret = append(ret, voterId)
		}
	}
	return ret
}

// Blocks until the vote session reaches the expected majority
func (s *voteSession) Wait() bool {
	select {
	case <-s.doneChan:
		return true
	case <-s.exitChan:
		return false
	}
}
