/*
*

	Contains the client-server protocol along with utility logging functions.
*/
package part1

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Message struct {
	SrcId int  // Source ID of message
	Data string // Message data
}

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
	for {
		select {
		case msg, ok := <-c.RecvChan:
			if !ok {
				// server closed channel
				log.Printf("C%d | QUIT\n", c.Id)
				return
			}
			log.Printf("C%d | RECV from SERVER | %v\n", c.Id, msg.Data)
		case <-time.After(c.SendIntv):
			data := fmt.Sprintf("C%dMSG%d", c.Id, c.Counter)
			c.Counter++
			c.SendChan <- Message{c.Id, data}
			log.Printf("C%d | SEND to SERVER | %v\n", c.Id, data)
		}
	}
}

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

func (s *Server) ConnectClient(clientId int, serverToClientChan chan<- Message, clientToServerChan <-chan Message) {
	s.SendChans = append(s.SendChans, serverToClientChan)

	// Start goroutine to automatically pass messages from clientToServerChan to recv channel
	go func(cTSChan <-chan Message) {
		for {
			msg, ok := <-cTSChan
			if !ok {
				log.Printf("Client %d closed connection.\n", msg.SrcId)
				return
			}

			s.RecvChan <- msg
		}
	}(clientToServerChan)
}

func (s *Server) Run() {
	for {
		select {
		case msg := <-s.RecvChan:
			// received message
			log.Printf("SERVER | RECV from C%d | %v", msg.SrcId, msg.Data)

			if rand.Float32() >= s.DropChance {
				// drop message
				continue
			}

			// forward message
			for clientId, clientSendChan := range(s.SendChans) {
				if clientId == msg.SrcId {
					continue
				}
				log.Printf("SERVER | SEND to C%d | %v", clientId, msg.Data)
				clientSendChan <- msg
			}
		case <-s.QuitChan:
			log.Println("SERVER | QUIT")
			
			// Close all channels
			for _, clientSendChan := range(s.SendChans) {
				close(clientSendChan)
			}

			close(s.RecvChan)
			return
		}
	}
}
