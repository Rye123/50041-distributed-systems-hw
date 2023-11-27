package main

type System struct {
	centralMgr *CentralManager
	nodes map[NodeId](*Node)
	nodeExits map[NodeId](chan bool)
	centralExit chan bool
}

func NewSystem(nodeCount int) *System {
	centralPort := NodePort{NodeId(0), make(chan Message)}
	nodePorts := make(map[NodeId]NodePort)
	nodeExits := make(map[NodeId](chan bool))
	exit := make(chan bool)
	for node_id := 1; node_id <= nodeCount; node_id++ {
		nodeId := NodeId(node_id)
		nodeExit := make(chan bool)
		nodePorts[nodeId] = NodePort{nodeId, make(chan Message)}
		nodeExits[nodeId] = nodeExit
	}

	cm := NewCentralManager(centralPort, nodePorts, exit)
	nodes := make(map[NodeId](*Node))
	for nodeId := range nodePorts {
		nodes[nodeId] = NewNode(nodeId, centralPort, nodePorts, nodeExits[nodeId])
	}

	return &System{cm, nodes, nodeExits, exit}
}

func (s *System) Init() {
	go s.centralMgr.Listen()
	for _, node := range s.nodes {
		go node.Listen()
	}
}

func (s *System) Exit() {
	for _, node := range s.nodes {
		node.exit <- true
	}
	s.centralMgr.exit <- true
}

func (s *System) NodeRead(nodeId NodeId, pageId PageId) string {
	return s.nodes[nodeId].ClientRead(pageId)
}

func (s *System) NodeWrite(nodeId NodeId, pageId PageId, data string) {
	s.nodes[nodeId].ClientWrite(pageId, data)
}


func main() {
	
}
