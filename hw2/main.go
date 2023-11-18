package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"fmt"
)



func main() {
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node), 0)
	// nodes[0] = nodetypes.NewNaiveNode(0, []int{0, 1, 2}, sm)
	// nodes[1] = nodetypes.NewNaiveNode(1, []int{0, 1, 2}, sm)
	// nodes[2] = nodetypes.NewNaiveNode(2, []int{0, 1, 2}, sm)
	endpoints := []nodetypes.LamportNodeEndpoint{
		nodetypes.NewLamportNodeEndpoint(0),
		nodetypes.NewLamportNodeEndpoint(1),
		nodetypes.NewLamportNodeEndpoint(2),
	}
	nodes[0] = nodetypes.NewLamportNode(0, endpoints, sm)
	nodes[1] = nodetypes.NewLamportNode(1, endpoints, sm)
	nodes[2] = nodetypes.NewLamportNode(2, endpoints, sm)
	
	o := NewOrchestrator(nodes, sm)
	o.Init()
	defer o.Shutdown()

	go func() {
		err := o.NodeEnter(0); if (err != nil) {fmt.Printf("Err: %v\n", err)}
		err = o.NodeExit(0); if err != nil {fmt.Printf("Err: %v\n", err)}
	}()

	go func() {
		err := o.NodeEnter(1); if (err != nil) {fmt.Printf("Err: %v\n", err)}
		err = o.NodeExit(1); if err != nil {fmt.Printf("Err: %v\n", err)}
	}()

	fmt.Scanf("%d")
}
