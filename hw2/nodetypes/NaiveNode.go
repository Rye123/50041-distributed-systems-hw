package nodetypes

type NaiveNode struct {
	nodeId int
	clock ClockVal
	smPtr *SharedMemory
}

func NewNaiveNode(nodeId int, nodeLs []int, sm *SharedMemory) *NaiveNode {
	return &NaiveNode{nodeId, ClockVal(0), sm}
}

func (n *NaiveNode) Init() (err error) { return nil }

func (n *NaiveNode) Shutdown() (err error) { return nil }

func (n *NaiveNode) AcquireLock() {
	n.clock++
	n.smPtr.EnterCS(n.nodeId, n.clock)
}

func (n *NaiveNode) ReleaseLock() {
	n.clock++
	n.smPtr.ExitCS(n.nodeId)
}
