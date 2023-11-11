package nodetypes

import (
	"1005129_RYAN_TOH/hw2/clock"
)

type LamportNode struct {
	nodeId int
	clock clock.ClockVal
	smPtr *SharedMemory
	queue *pqueue
}

func NewLamportNode(nodeId int, nodeIds []int, sm *SharedMemory) *LamportNode {
	return &LamportNode{nodeId, clock.NewClockVal(nodeIds), sm, newPQueue}
}

func (n *LamportNode) AcquireLock() {
	// Make a request with timestamp, and add req to queue

	// Block until we've received responses from all nodes AND request is at head of queue

	// Enter the CS
	n.smPtr.EnterCS(n.nodeId, n.clock.Clone())
}

func (n *LamportNode) ReleaseLock() {
	// Exit the CS
	n.smPtr.ExitCS(n.nodeId)
	
	// Pop head of queue

	// Broadcast request with timestamp: RELEASE

}
