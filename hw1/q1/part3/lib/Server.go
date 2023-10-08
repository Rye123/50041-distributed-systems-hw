package lib

import (
	"log"
	"math/rand"
	"sync"
)

type Server struct {
	Id         int
	Clock      ClockVal
	RecvChan   chan Message             // Joint channel to receive messages from clients
	SendChans  map[int](chan<- Message) // Client receive channels, used for broadcast
	DropChance float32                  // Chance of server dropping a message
	QuitChan   <-chan bool
	clientWg sync.WaitGroup
}

// Initialise a new server.
func NewServer(nodeIds []int, recvChan chan Message, dropChance float32, quitChan <-chan bool) Server {
	log.Printf("Server: Drop Chance: %v", dropChance)
	return Server{-1, NewClockVal(nodeIds), recvChan, make(map[int](chan<- Message)), dropChance, quitChan, sync.WaitGroup{}}
}

// Sends a given message to the given clientId.
func (s *Server) Send(clientId int, msg Message) {
	log.Printf("Server: SEND to C%d  : %v", clientId, msg.Data)
	s.Clock = s.Clock.Increment(s.Id, 1)

	// Ensure timestamp of message is set
	msg.Timestamp = s.Clock

	// Send the message.
	s.SendChans[clientId] <- msg
}

// Handles the reception of a given message.
func (s *Server) Handle(msg Message) {
	log.Printf("Server: RECV from C%d: %v", msg.SrcId, msg.Data)

	// Check for potential causality violation
	if s.Clock.Compare(msg.Timestamp) > 1 {
		// local clock > received clock
		log.Printf("Server: Dropping msg from C%d (%v) due to potential causality violation", msg.SrcId, msg.Data)
		return
	}

	// Update clock based on timestamp, and add one for ID due to recv event
	s.Clock = MaxClockValue(s.Clock, msg.Timestamp).Increment(s.Id, 1)

	// Random Drop
	if rand.Float32() < s.DropChance {
		log.Printf("Server: DROP message: %v", msg.Data)
		return
	}

	// Forward message through broadcast
	for clientId := range s.SendChans {
		if clientId == msg.SrcId {
			continue
		}
		s.Send(clientId, msg)
	}
}

// Connect a given client
func (s *Server) ConnectClient(clientId int, serverToClientChan chan<- Message, clientToServerChan <-chan Message) {
	s.SendChans[clientId] = serverToClientChan

	// Set goroutine to forward messages from clientToServerChan to joint channel
	go func(clientSendChan <-chan Message) {
		s.clientWg.Add(1)
		defer s.clientWg.Done()

		for {
			msg, ok := <-clientSendChan
			if !ok {
				return
			}

			// Pass message to actual centralised channel
			s.RecvChan <- msg
		}
	}(clientToServerChan)
}

// Runs the server.
func (s *Server) Run() {
	for {
		select {
		case msg := <-s.RecvChan:
			// Received a message
			s.Handle(msg)
		case <-s.QuitChan:
			log.Println("Server: QUIT")

			// Close all sending channels
			for clientId := range s.SendChans {
				close(s.SendChans[clientId])
			}

			// Close receiving channel only after all clientToServer channels have closed
			log.Println("Server: Waiting for client channels to close...")
			s.clientWg.Wait()
			close(s.RecvChan)
			log.Println("Server: QUIT SUCCESS")
			return
		}
	}
}
