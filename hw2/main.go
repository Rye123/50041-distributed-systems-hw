package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
)



func main() {
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodes[0] = nodetypes.NewNaiveNode(0, []int{0, 1, 2}, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, []int{0, 1, 2}, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, []int{0, 1, 2}, sm)
	o := NewOrchestrator(nodes, sm)
	o.NodeEnter(0)
	o.NodeExit(0)
	o.NodeEnter(1)
	o.NodeExit(1)
	o.NodeEnter(2)
	o.NodeEnter(1)
}
