/**
  Q1: Simulate the behaviour of both the server and the registered clients via GO routines.
*/

package main

import (
	"fmt"
	"log"
	"time"
	"math/rand"
)

const NUM_CLIENTS = 10
const SEND_INTV_FLOOR = 3
const SEND_INTV_CEIL = 15


// Message to be sent between clients and the server
type Message struct {
	// Source ID
	id int

	// Message data
	data string
}

type Node interface {
	Run()
}

type Client struct {
	// ID of the client
	id int
	// Channel that receives messages from server
	recvChan <-chan Message
	// Channel to send messages to server
	sendChan chan<- Message
	// Counter, used as part of the message.
	counter int
}

// Creates a new client
func NewClient(clientId int, recvChan chan Message, sendChan chan Message) Client {
	return Client{clientId, recvChan, sendChan, 0}
}

func (c *Client) Run() {
	log.Printf("Client %d running.\n", c.id)
	for {
		sendIntv := rand.Intn(SEND_INTV_CEIL - SEND_INTV_FLOOR + 1) + SEND_INTV_FLOOR
		select {
		case msg, ok := <-c.recvChan:
			if !ok {
				log.Printf("Client %d detected server close, exiting.", c.id)
				close(c.sendChan)
				return
			}
			log.Printf("Client %d received: %v\n", c.id, msg.data)
		case <-time.After(time.Second * time.Duration(sendIntv)):
			data := fmt.Sprintf("Client%dMsg%d", c.id, c.counter)
			c.counter++
			c.sendChan <- Message{c.id, data}
			log.Printf("Client %d sent: %v\n", c.id, data)
		}
	}
}

type Server struct {
	clientSendChans []chan<- Message // all individual channels to send to clients
	quitChan chan bool
}

func NewServer(quitChan chan bool) Server {
	return Server{
		make([]chan<- Message, 0),
		quitChan,
	}
}

func (s *Server) ConnectClient(clientId int, serverToClientChan chan<- Message, clientToServerChan <-chan Message) {
	s.clientSendChans = append(s.clientSendChans, serverToClientChan)

	// Start goroutine to listen and pass new messages to main channel
	go func(clientToServerChan <-chan Message) {
		for {
			msg, ok := <-clientToServerChan
			if !ok {
				log.Printf("Server: Client %d closing.\n", clientId)
			}
			
			// Handle message
			log.Printf("Server received from client %d: %v\n", clientId, msg.data)

			// Forward messages to all but the one who sent
			for clientId, clientSendChan := range(s.clientSendChans) {
				if clientId == msg.id {
					continue
				}
				log.Printf("Server forwarded to client %d: %v\n", clientId, msg.data)
				clientSendChan <- msg
			}
		}
	}(clientToServerChan)
	log.Printf("Server: Connected new client: %d\n", clientId)
}

func (s *Server) Run() {
	for {
		quit := <-s.quitChan
		if quit {
			log.Printf("Server: Quitting.")
			// Close all serverToClient channels
			for _, clientSendChan := range(s.clientSendChans) {
				close(clientSendChan)
			}
			return
		}
	}
}

func main() {
	quit := make(chan bool)
	defer func() { quit <- true }()
	
	// Initialise clients and servers
	server := NewServer(quit)
	clients := make([]Client, 0)
	for clientId := 0; clientId < NUM_CLIENTS; clientId++ {
		clientId := clientId
		serverToClientChan := make(chan Message)
		clientToServerChan := make(chan Message)
		clients = append(clients, NewClient(clientId, serverToClientChan, clientToServerChan))
		server.ConnectClient(clientId, serverToClientChan, clientToServerChan)
	}

	// Start server and clients
	go server.Run()
	for _, client := range(clients) {
		client := client
		go client.Run()
	}

	for {
		var in string
		_, err := fmt.Scanln(&in)
		if err != nil || in == "q" || in == "Q" {
			log.Println("COMMAND: EXIT")
			return
		}
	}
}
