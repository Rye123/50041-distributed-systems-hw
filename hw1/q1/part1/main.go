package main

import (
	"fmt"
	"1005129_RYAN_TOH/hw1/q1/part1/lib"
	"math/rand"
	"sync"
)

const CLIENT_COUNT = 10
const SERVER_DROP_CHANCE = 0.5

// To set the random delay of client sending messages (in milliseconds)
const CLIENT_DELAY_FLOOR = 500
const CLIENT_DELAY_CEIL = 5000

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
	fmt.Println("Initialising system. To safely exit, press ENTER.")
	var wg sync.WaitGroup

	quit := make(chan bool)
	defer func() {
		fmt.Println("Stopping goroutines...")
		quit <- true
		wg.Wait()
		fmt.Println("All goroutines stopped.")
	}()

	serverRecvChan := make(chan lib.Message)
	server := lib.NewServer(serverRecvChan, SERVER_DROP_CHANCE, quit)
	clients := make([]lib.Client, 0)

	for i := 0; i < CLIENT_COUNT; i++ {
		clientId := i
		sendIntvMS := IntInRange(CLIENT_DELAY_FLOOR, CLIENT_DELAY_CEIL)
		clientSendChan := make(chan lib.Message)
		clientRecvChan := make(chan lib.Message)
		clients = append(clients, lib.NewClient(clientId, clientRecvChan, clientSendChan, sendIntvMS))

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
