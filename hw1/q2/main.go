package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Rye123/50041-distributed-systems-hw/hw1/q2/lib"
)

const DEFAULT_SEND_INTV = 5 * time.Second
const DEFAULT_TIMEOUT = 1 * time.Second

func InitNodes(nodeCount int) [](*lib.Node) {
	log.Printf("SYSTEM: Initialising with %d nodes", nodeCount)
	nodes := make([](*lib.Node), 0)

	for i := 0; i < nodeCount; i++ {
		nodeId := lib.NodeId(i)
		nodes = append(nodes, lib.NewNode(nodeId, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT, false, nodeCount))
	}

	// Get endpoints of each node
	endpoints := make([]lib.NodeEndpoint, 0)
	for _, node := range nodes {
		endpoints = append(endpoints, node.Endpoint)
	}

	// Initialise nodes with endpoints
	for _, node := range nodes {
		go node.Initialise(endpoints)
	}
	return nodes
}

func CleanupNodes(nodes [](*lib.Node)) {
	log.Printf("SYSTEM: Cleaning up nodes...")
	var exitWg sync.WaitGroup

	for _, node := range nodes {
		exitWg.Add(1)
		go func(n *lib.Node) {
			defer exitWg.Done()
			n.Exit()
		}(node)
	}
	log.Printf("SYSTEM: Node cleanup complete.")
}

func main() {
	nodes := InitNodes(10)
	defer CleanupNodes(nodes)
	counter := 0

	for {
		// Perform a random action
		randInt := rand.Int31n(4)
		fmt.Printf("\n---\n\n")
		switch randInt {
		case 0:
			// Update node value
			for i := len(nodes) - 1; i > 0; i-- {
				if nodes[i].IsAlive {
					value := fmt.Sprintf("Msg%d", counter)
					counter++
					fmt.Printf("SYSTEM: Updating N%d with value %v\n", nodes[i].Id, value)
					nodes[i].PushUpdate(value)
					break
				}
			}
		case 1, 2:
			// Kill or reboot random node
			randNodeId := lib.NodeId(rand.Int31n(int32(len(nodes))))
			if nodes[randNodeId].IsAlive {
				fmt.Printf("SYSTEM: Killing N%d.\n", randNodeId)
				nodes[randNodeId].Kill()
			} else {
				fmt.Printf("SYSTEM: Rebooting N%d.\n", randNodeId)
				nodes[randNodeId].Restart()
			}
		case 3:
			// Kill coordinator
			for i := len(nodes) - 1; i > 0; i-- {
				if nodes[i].IsAlive {
					fmt.Printf("SYSTEM: Killing N%d.\n", nodes[i].Id)
					nodes[i].Kill()
					break
				}
			}
		}
		fmt.Printf("\n---\n")
		time.Sleep(DEFAULT_SEND_INTV + DEFAULT_TIMEOUT)
	}
}
