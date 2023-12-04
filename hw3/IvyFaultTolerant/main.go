package main

type System struct {
	cm1 *CentralManager
	cm2 *CentralManager
	nodes map[NodeId](*Node)
	nodeExits map[NodeId](chan bool)
	centralExit chan bool
}

func NewSystem(nodeCount int) *System {
	cm1Port := NodePort{-1, make(chan Message)}
	cm1InternalPort := InternalPort{-1, make(chan CMMessage)}
	cm2Port := NodePort{-2, make(chan Message)}
	cm2InternalPort := InternalPort{-2, make(chan CMMessage)}
	
	nodePorts := make(map[NodeId]NodePort)
	nodeExits := make(map[NodeId](chan bool))
	
	exit := make(chan bool)
	
	for node_id := 1; node_id <= nodeCount; node_id++ {
		nodeId := NodeId(node_id)
		nodeExit := make(chan bool)
		nodePorts[nodeId] = NodePort{nodeId, make(chan Message)}
		nodeExits[nodeId] = nodeExit
	}

	cm1 := NewCentralManager(NodeId(-1), cm1Port, cm1InternalPort, cm2InternalPort, nodePorts, exit)
	cm2 := NewCentralManager(NodeId(-2), cm2Port, cm2InternalPort, cm1InternalPort, nodePorts, exit)

	nodes := make(map[NodeId](*Node))
	cmPorts := map[NodeId]NodePort{NodeId(-1): cm1Port, NodeId(-2): cm2Port}
	for nodeId := range nodePorts {
		nodes[nodeId] = NewNode(nodeId, cmPorts, nodePorts, nodeExits[nodeId])
	}

	return &System{cm1, cm2, nodes, nodeExits, exit}
}

func (s *System) Init() {
	go s.cm1.Listen()
	go s.cm2.Listen()
	for _, node := range s.nodes {
		go node.Listen()
	}
}

func (s *System) Exit() {
	for _, node := range s.nodes {
		node.exit <- true
	}
	s.cm1.exit <- true
	s.cm2.exit <- true
}

func (s *System) NodeRead(nodeId NodeId, pageId PageId) string {
	return s.nodes[nodeId].ClientRead(pageId)
}

func (s *System) NodeWrite(nodeId NodeId, pageId PageId, data string) {
	s.nodes[nodeId].ClientWrite(pageId, data)
}


func main() {
	
}
