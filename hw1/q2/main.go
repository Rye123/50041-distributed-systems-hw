package main

import (
	"fmt"
	"time"

	"github.com/Rye123/50041-distributed-systems-hw/hw1/q2/lib"
)

const DEFAULT_SEND_INTV = 5 * time.Second
const DEFAULT_TIMEOUT = 1 * time.Second

func main() {
	fmt.Println("Initialising")
	nodePtrs := [](*lib.Node){
		lib.NewNode(0, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT),
		lib.NewNode(1, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT),
		lib.NewNode(2, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT),
	}

	nodeLs := make([]lib.Node, 0)
	for _, nodePtr := range(nodePtrs) {
		nodeLs = append(nodeLs, *nodePtr)
	}
	
	go nodeLs[0].Initialise(nodeLs)
	go nodeLs[1].Initialise(nodeLs)
	go nodeLs[2].Initialise(nodeLs)
	
	fmt.Println("Initialised. Press ENTER to trigger update of data")
	fmt.Scanf("%d")
	nodeLs[2].UpdateData("Hello world")
	
	fmt.Println("Updated. Press ENTER to trigger update of data")
	fmt.Scanf("%d")
	nodeLs[2].UpdateData("Goodbye cruel world")
	
	fmt.Println("Updated. Press ENTER to trigger killing of node 2")
	fmt.Scanf("%d")
	nodeLs[2].Kill()
	
	fmt.Println("Killed. Press ENTER to trigger restart of node 2")
	fmt.Scanf("%d")

	nodeLs[2].Restart()

	fmt.Scanf("%d")
}
