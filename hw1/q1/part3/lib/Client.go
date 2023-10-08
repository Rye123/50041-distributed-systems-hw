package lib

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"
)

type Client struct {
	Id                       int
	Clock                    ClockVal
	RecvChan                 <-chan Message
	SendChan                 chan<- Message
	SendIntv                 time.Duration // Time between sending messages
	Counter                  int           // Counter used to distinguish messages from each other.
	RecvdMsgs                []Message     // Contains all received messages
	CausalityViolationChance float32
}

// Initialise a new client
func NewClient(clientId int, nodeIds []int, recvChan <-chan Message, sendChan chan<- Message, sendIntvMS int, causalityViolationChance float32) Client {
	sendIntv := time.Millisecond * time.Duration(sendIntvMS)
	log.Printf("C%d, Send Interval: %d milliseconds, Causality Violation Chance: %v", clientId, sendIntvMS, causalityViolationChance)
	return Client{clientId, NewClockVal(nodeIds), recvChan, sendChan, sendIntv, 0, make([]Message, 0), causalityViolationChance}
}

// Sends a given message along SendChan
func (c *Client) Send(msg Message) {
	// Random chance of a causality violation
	if rand.Float32() < c.CausalityViolationChance {
		// Since Go channels have no chance of receiving messages out-of-order,
		// we simulate it by creating two messages with different clocks,
		// and sending the later one before the earlier one.

		log.Printf("C%d: SIMULATE CAUSALITY VIOLATION", c.Id)
		c.Clock = c.Clock.Increment(c.Id, 1)
		m1 := Message{c.Id, msg.Data + "-1", c.Clock.Clone()}
		c.Clock = c.Clock.Increment(c.Id, 1)
		m2 := Message{c.Id, msg.Data + "-2", c.Clock.Clone()}

		log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, m1.Data)
		log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, m2.Data)
		c.SendChan <- m2
		c.SendChan <- m1
	} else {
		log.Printf("C%d: SEND to SERVER  : %v\n", c.Id, msg.Data)
		c.Clock = c.Clock.Increment(c.Id, 1)

		// Ensure timestamp of message is set
		msg.Timestamp = c.Clock.Clone()

		// Send the message.
		c.SendChan <- msg
	}

}

// Handles the reception of a given message.
func (c *Client) Handle(msg Message) {
	log.Printf("C%d: RECV from SERVER: %v\n", c.Id, msg.Data)

	// Check for potential causality violation
	if c.Clock.Compare(msg.Timestamp) > 1 {
		// local clock > received clock
		log.Printf("C%d: Dropping msg from SERVER (%v) due to potential causality violation", c.Id, msg.Data)
		return
	}

	// Update clock based on timestamp, adding one to self ID due to recv event
	c.Clock = MaxClockValue(c.Clock, msg.Timestamp).Increment(c.Id, 1)

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
		log.Printf("C%d: Received Message Order: %v", c.Id, c.ReportMessages())
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
			msg := Message{c.Id, fmt.Sprintf("C%d-MSG%d", c.Id, c.Counter), c.Clock.Clone()}
			c.Counter++
			c.Send(msg)
		}
	}
}
