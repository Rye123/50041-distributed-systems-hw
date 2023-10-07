package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const CLIENT_COUNT = 20
const SERVER_DROP_CHANCE = 0.5

// To set the random delay of client sending messages (in milliseconds)
const CLIENT_DELAY_FLOOR = 500
const CLIENT_DELAY_CEIL  = 10000

// Message sent between server and client
type Message struct {
	SrcId int  // Source ID of message
	Data string // Message data
}

// Client sends on SendChan and receives on RecvChan.
type Client struct {
	Id int
	RecvChan <-chan Message  // Receive channel
	SendChan chan<- Message  // Send channel
	SendIntv time.Duration   // Time between sending messages
	Counter  int             // Counter, used as part of the message
}

func NewClient(clientId int, recvChan <-chan Message, sendChan chan<- Message, sendIntv time.Duration) Client {
	return Client{clientId, recvChan, sendChan, sendIntv, 0}
}

func (c *Client) Run() {
	quit := make(chan bool)
	
	// Initialise separate sending goroutine
	go func(quitChan chan bool) {
		for {
			select {
			case <-quit:
				return
			case <-time.After(c.SendIntv):
				data := fmt.Sprintf("C%d-MSG%d", c.Id, c.Counter)
				c.Counter++
				c.SendChan <- Message{c.Id, data}
				log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, data)
			}
		}
	}(quit)

	// Listen for messages
	for {
		msg, ok := <-c.RecvChan
		if !ok {
			// server closed channel
			log.Printf("C%d: QUIT\n", c.Id)
			quit <- true
			return
		}
		log.Printf("C%d: RECV from SERVER: %v\n", c.Id, msg.Data)
	}
}

// Server receives on RecvChan, and sends to a client i with SendChans[i].
type Server struct {
	Id int
	RecvChan chan Message    // Joint channel to receive messages from clients
	SendChans []chan<- Message // Client receive channels, used for broadcast
	DropChance float32         // Chance of server dropping a message
	QuitChan <-chan bool
}

func NewServer(recvChan chan Message, dropChance float32, quitChan <-chan bool) Server {
	return Server{-1, recvChan, make([]chan<- Message, 0), dropChance, quitChan}
}

func (s *Server) ConnectClient(clientId int, serverToClientChan chan<- Message) {
	s.SendChans = append(s.SendChans, serverToClientChan)
}

func (s *Server) Run() {
	for {
		select {
		case msg := <-s.RecvChan:
			// received message
			log.Printf("Server: RECV from C%d: %v", msg.SrcId, msg.Data)

			if rand.Float32() <  s.DropChance {
				// drop message
				log.Printf("Server: DROP message: %v", msg.Data)
				continue
			}

			// forward message
			for clientId, clientSendChan := range(s.SendChans) {
				if clientId == msg.SrcId {
					continue
				}
				log.Printf("Server: SEND to C%d  : %v", clientId, msg.Data)
				clientSendChan <- msg
			}
		case <-s.QuitChan:
			log.Println("Server: QUIT")
			
			// Close all channels
			for _, clientSendChan := range(s.SendChans) {
				close(clientSendChan)
			}

			close(s.RecvChan)
			return
		}
	}
}

// Returns a random integer in the range [floor, ceil].
func IntInRange(floor int, ceil int) int {
	if floor == ceil {
		return floor
	}
	if floor > ceil {
		panic("IntInRange floor > ceil")
	}
	return rand.Intn(ceil - floor) + floor
}

func main() {
	quit := make(chan bool)
	defer func() {
		quit <- true
	}();

	serverRecvChan := make(chan Message)
	server := NewServer(serverRecvChan, SERVER_DROP_CHANCE, quit)
	log.Printf("Server: DROP CHANCE: %v", SERVER_DROP_CHANCE)
	clients := make([]Client, 0)

	for i := 0; i < CLIENT_COUNT; i++ {
		clientId := i
		delay := IntInRange(CLIENT_DELAY_FLOOR, CLIENT_DELAY_CEIL)
		clientSendIntv := time.Millisecond * time.Duration(delay)
		clientRecvChan := make(chan Message)
		clients = append(clients, NewClient(clientId, clientRecvChan, serverRecvChan, clientSendIntv))
		log.Printf("C%d: DELAY: %d milliseconds", clientId, delay)

		server.ConnectClient(clientId, clientRecvChan)
	}

	// Start clients and server
	go server.Run()
	for _, client := range(clients) {
		client := client
		go client.Run()
	}

	// Wait for end
	fmt.Scanf("%s")
}
