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
	nodes := [](*lib.Node){
		lib.NewNode(0, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
		lib.NewNode(1, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
		lib.NewNode(2, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
		lib.NewNode(3, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
		lib.NewNode(4, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
		lib.NewNode(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false),
	}

	endpoints := make([]lib.NodeEndpoint, 0)
	for _, node := range(nodes) {
		endpoints = append(endpoints, node.Endpoint)
	}
	
	// Initialise nodes
	for _, node := range(nodes) {
		go node.Initialise(endpoints)
	}

	defer func() {
		for _, node := range(nodes) {
			go node.Exit()
		}
	}()
	
	fmt.Println("All nodes initialised.")

	// Simulate updating of data
	str := "asdf"
	time.Sleep(5 * time.Second)
	nodes[5].PushUpdate(str)

	fmt.Println("Press ENTER to trigger killing of node 5")

	fmt.Scanf("%d")
	nodes[5].Kill()
	
	fmt.Println("Killed. Press ENTER to trigger restart of node 5")
	fmt.Scanf("%d")
	nodes[5].Restart()

	fmt.Println("Restarted. Press ENTER to stop.")
	fmt.Scanf("%d")
}
