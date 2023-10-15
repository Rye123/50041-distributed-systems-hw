package main

import (
	"fmt"
	"github.com/Rye123/50041-distributed-systems-hw/hw1/q1/part3/lib"
	"math/rand"
	"sync"
)

const CLIENT_COUNT = 20
const SERVER_DROP_CHANCE = 0.5
const CAUSALITY_VIOLATION_CHANCE = 0.5

// To set the random delay of client sending messages (in milliseconds)
const CLIENT_DELAY_FLOOR = 1000
const CLIENT_DELAY_CEIL = 10000

// Returns a random integer in the range [floor, ceil].
func IntInRange(floor int, ceil int) int {
	if floor == ceil {
		return floor
	}
	if floor > ceil {
		panic("IntInRange floor > ceil")
	}
	return rand.Intn(ceil-floor) + floor
}

func main() {
	fmt.Println("Initialising system. To safely exit and print the relevant messages, press ENTER.")
	var wg sync.WaitGroup

	quit := make(chan bool)
	defer func() {
		fmt.Println("Stopping goroutines...")
		quit <- true
		wg.Wait()
		fmt.Println("All goroutines stopped.")
	}()

	// Generate client IDs
	nodeIds := make([]int, CLIENT_COUNT)
	nodeIds = append(nodeIds, -1) // Hardcoded server ID
	for nodeId := 0; nodeId < CLIENT_COUNT; nodeId++ {
		nodeIds = append(nodeIds, nodeId)
	}

	serverRecvChan := make(chan lib.Message)
	server := lib.NewServer(nodeIds, serverRecvChan, SERVER_DROP_CHANCE, quit)
	clients := make([]lib.Client, 0)

	for i := 0; i < CLIENT_COUNT; i++ {
		clientId := i
		sendIntvMS := IntInRange(CLIENT_DELAY_FLOOR, CLIENT_DELAY_CEIL)
		clientSendChan := make(chan lib.Message)
		clientRecvChan := make(chan lib.Message)
		clients = append(clients, lib.NewClient(clientId, nodeIds, clientRecvChan, clientSendChan, sendIntvMS, CAUSALITY_VIOLATION_CHANCE))

		server.ConnectClient(clientId, clientRecvChan, clientSendChan)
	}

	// Start clients and server
	go func() {
		wg.Add(1)
		defer wg.Done()
		server.Run()
	}()
	for _, client := range clients {
		client := client
		go func(c lib.Client) {
			wg.Add(1)
			defer wg.Done()
			client.Run()
		}(client)
	}

	// Wait for end
	fmt.Scanf("%s")
}