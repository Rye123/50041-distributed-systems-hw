package lib

import (
	"errors"
	"fmt"
	"time"
)

type Orchestrator struct {
	Nodes map[NodeId](*Node)
}

func NewOrchestrator(nodeCount int, sendIntv, timeout time.Duration) *Orchestrator {
	nodes := make(map[NodeId](*Node))

	for nodeId := 0; nodeId < nodeCount; nodeId++ {
		nodeId := NodeId(nodeId)
		nodes[nodeId] = NewNode(nodeId, sendIntv, timeout, false, nodeCount)
	}

	return &Orchestrator{nodes}
}

// Initialise system, returns only after election completed
func (o *Orchestrator) Initiate() {
	endpoints := make([]NodeEndpoint, 0)
	for nodeId := range(o.Nodes) {
		endpoints = append(endpoints, o.Nodes[nodeId].Endpoint)
	}
	
	// Initialise nodes
	for nodeId := range(o.Nodes) {
		go o.Nodes[nodeId].Initialise(endpoints)
	}
}

func (o *Orchestrator) GetValues() map[NodeId]string {
	coordValues := make(map[NodeId]string)
	for nodeId := range(o.Nodes) {
		// Check first if the node is actually alive, since dead nodes won't be updated
		if !o.Nodes[nodeId].IsAlive {
			continue
		}
		coordValues[nodeId] = o.Nodes[nodeId].Data
	}
	return coordValues
}

// Returns the overall value. Throws an error if the values are not the same.
func (o *Orchestrator) GetValue() (string, error) {
	if len(o.Nodes) == 0 {
		panic("No nodes to get value from")
	}
	
	value := ""
	readFirstLiveNode := false
	
	for _, nodeValue := range(o.GetValues()) {
		if value != nodeValue {
			// either we're at the first live node, or value is different
			if readFirstLiveNode {
				// then value is different
				return "", errors.New(fmt.Sprintf("No single value: Has \"%s\" and \"%s\".", nodeValue, value))
			} else {
				// we're at the first live node
				value = nodeValue
			}
		}
	}

	return value, nil
}

func (o *Orchestrator) UpdateNodeValue(id NodeId, value string, force bool) {
	if force {
		o.Nodes[id].Data = value
	} else {
		o.Nodes[id].PushUpdate(value)
	}
}

func (o *Orchestrator) KillNode(id NodeId) {
	o.Nodes[id].Kill()
}

func (o *Orchestrator) RestartNode(id NodeId) {
	if o.Nodes[id].IsAlive {
		panic("Tried to start an alive node")
	}
	o.Nodes[id].Restart()
}

func (o *Orchestrator) HasOngoingElection() bool {
	for nodeId := range(o.Nodes) {
		if o.Nodes[nodeId].ongoingElectionId != "" {
			return true
		}
	}
	return false
}

func (o *Orchestrator) BlockTillElectionStart(maxPolls int, pollTick time.Duration) error {
	polls := 0
	for polls < maxPolls {
		if !o.HasOngoingElection() {
			time.Sleep(pollTick)
			polls++
		} else {
			break
		}
	}

	if polls == maxPolls {
		return errors.New(fmt.Sprintf("Election not settled after %d polls.", maxPolls))
	}

	return nil
}

func (o *Orchestrator) BlockTillElectionDone(maxPolls int, pollTick time.Duration) error {
	polls := 0
	for polls < maxPolls {
		if o.HasOngoingElection() {
			time.Sleep(pollTick)
			polls++
		} else {
			break
		}
	}

	if polls == maxPolls {
		return errors.New(fmt.Sprintf("Election not settled after %d polls.", maxPolls))
	}

	return nil
}

func (o *Orchestrator) GetCoordinatorIds() map[NodeId]NodeId {
	coordIds := make(map[NodeId]NodeId)
	for nodeId := range(o.Nodes) {
		// Check first if the node is actually alive, since we set the coordinator ID of dead nodes to be -1
		if !o.Nodes[nodeId].IsAlive {
			continue
		}
		coordIds[nodeId] = o.Nodes[nodeId].CoordinatorId
	}
	return coordIds
}

// Returns the coordinator ID after election has settled.
// Blocks if election has not been settled, gives error after 5 seconds.
func (o *Orchestrator) GetCoordinatorId(maxPolls int, pollTick time.Duration) (NodeId, error) {
	err := o.BlockTillElectionDone(maxPolls, pollTick)
	if err != nil {
		return -1, errors.New("Election not settled.")
	}
	
	// Ensure coordId is the same
	coordId := NodeId(-1)
	readFirstLiveNode := false
	for i, nodeId := range(o.GetCoordinatorIds()) {
		if readFirstLiveNode {
			if nodeId != coordId {
				return -1, errors.New(fmt.Sprintf("Election settled, but no single coordinatorId (Has C%d (from N%d) and N%d.", nodeId, i, coordId))
			}
		} else {
			coordId = nodeId
		}
	}

	return coordId, nil
}

func (o *Orchestrator) Exit() {
	for nodeId := range(o.Nodes) {
		o.Nodes[nodeId].Exit()
	}
}
