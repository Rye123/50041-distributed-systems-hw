package nodetypes

import (
	"1005129_RYAN_TOH/hw2/clock"
)

type NaiveNode struct {
	nodeId int
	clock clock.ClockVal
	smPtr *SharedMemory
}

func NewNaiveNode(nodeId int, nodeIds []int, sm *SharedMemory) *NaiveNode {
	return &NaiveNode{nodeId, clock.NewClockVal(nodeIds), sm}
}

func (n *NaiveNode) AcquireLock() {
	n.clock = n.clock.Increment(n.nodeId, 1)
	n.smPtr.EnterCS(n.nodeId, n.clock.Clone())
}

func (n *NaiveNode) ReleaseLock() {
	n.clock = n.clock.Increment(n.nodeId, 1)
	n.smPtr.ExitCS(n.nodeId)
}
