package lib

type Orchestrator struct {
	Nodes map[NodeId](*Node)
}

func (o *Orchestrator) GetCoordinatorIds() map[NodeId]NodeId {
	coordIds := make(map[NodeId]NodeId)
	for nodeId := range(o.Nodes) {
		coordIds[nodeId] = o.Nodes[nodeId].CoordinatorId
	}
	return coordIds
}

func (o *Orchestrator) KillNode(id NodeId) {
	o.Nodes[id].Kill()
}
