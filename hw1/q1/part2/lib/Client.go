package lib

import (
	"fmt"
	"log"
	"sort"
	"time"
)

type Client struct {
	Id        int
	Clock     ClockVal
	RecvChan  <-chan Message
	SendChan  chan<- Message
	SendIntv  time.Duration // Time between sending messages
	Counter   int           // Counter used to distinguish messages from each other.
	RecvdMsgs []Message     // Contains all received messages
}

// Initialise a new client
func NewClient(clientId int, recvChan <-chan Message, sendChan chan<- Message, sendIntvMS int) Client {
	sendIntv := time.Millisecond * time.Duration(sendIntvMS)
	log.Printf("C%d, Send Interval: %d milliseconds", clientId, sendIntvMS)
	return Client{clientId, 0, recvChan, sendChan, sendIntv, 0, make([]Message, 0)}
}

// Sends a given message along SendChan
func (c *Client) Send(msg Message) {
	log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, msg.Data)
	c.Clock++

	// Ensure timestamp of message is set
	msg.Timestamp = c.Clock

	// Send the message.
	c.SendChan <- msg
}

// Handles the reception of a given message.
func (c *Client) Handle(msg Message) {
	log.Printf("C%d: RECV from SERVER: %v\n", c.Id, msg.Data)

	// Update clock based on timestamp, adding one due to recv event
	c.Clock = MaxClockValue(c.Clock, msg.Timestamp) + 1

	c.RecvdMsgs = append(c.RecvdMsgs, msg)
}

// Returns a string of all messages in order of timestamp.
func (c *Client) ReportMessages() string {
	sort.Slice(c.RecvdMsgs, func(i, j int) bool {
		return MessageLessThan(c.RecvdMsgs[i], c.RecvdMsgs[j])
	})

	output := fmt.Sprintf("[")
	for _, msg := range c.RecvdMsgs {
		output += msg.Data + ","
	}
	if len(c.RecvdMsgs) == 0 {
		output += "]"
	} else {
		output = output[:len(output)-1] + "]"
	}

	return output
}

// Runs the client
func (c *Client) Run() {
	sendTicker := time.NewTicker(c.SendIntv)
	defer func() {
		sendTicker.Stop()
		log.Printf("C%d: Total Order of Received Messages: %v", c.Id, c.ReportMessages())
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
			msg := Message{c.Id, fmt.Sprintf("C%d-MSG%d", c.Id, c.Counter), -1}
			c.Counter++
			c.Send(msg)
		}
	}
}
