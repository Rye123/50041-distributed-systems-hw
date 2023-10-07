package lib

import (
	"fmt"
	"log"
	"time"
)

type Client struct {
	Id       int
	RecvChan <-chan Message
	SendChan chan<- Message
	SendIntv time.Duration // Time between sending messages
	Counter  int           // Counter used to distinguish messages from each other.
}

// Initialise a new client
func NewClient(clientId int, recvChan <-chan Message, sendChan chan<- Message, sendIntvMS int) Client {
	sendIntv := time.Millisecond * time.Duration(sendIntvMS)
	log.Printf("C%d, Send Interval: %d milliseconds", clientId, sendIntvMS)
	return Client{clientId, recvChan, sendChan, sendIntv, 0}
}

// Sends a given message along SendChan
func (c *Client) Send(msg Message) {
	log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, msg.Data)
	c.SendChan <- msg
}

// Handles the reception of a given message.
func (c *Client) Handle(msg Message) {
	log.Printf("C%d: RECV from SERVER: %v\n", c.Id, msg.Data)
}

// Runs the client
func (c *Client) Run() {
	sendTicker := time.NewTicker(c.SendIntv)
	defer func() {
		sendTicker.Stop()
		log.Printf("C%d: QUIT", c.Id)
		close(c.SendChan)
	}()

	for {
		select {
		case msg, ok := <-c.RecvChan:
			if !ok {
				// Server closed channel
				return
			}
			c.Handle(msg)
		case <-sendTicker.C:
			msg := Message{c.Id, fmt.Sprintf("C%d-MSG%d", c.Id, c.Counter)}
			c.Counter++
			c.Send(msg)
		}
	}
}
