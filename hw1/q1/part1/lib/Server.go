package lib

import (
	"log"
	"math/rand"
)

type Server struct {
	Id         int
	RecvChan   chan Message             // Joint channel to receive messages from clients
	SendChans  map[int](chan<- Message) // Client receive channels, used for broadcast
	DropChance float32                  // Chance of server dropping a message
	QuitChan   <-chan bool
}

// Initialise a new server.
func NewServer(recvChan chan Message, dropChance float32, quitChan <-chan bool) Server {
	log.Printf("Server: Drop Chance: %v", dropChance)
	return Server{-1, recvChan, make(map[int](chan<- Message)), dropChance, quitChan}
}

// Sends a given message to the given clientId.
func (s *Server) Send(clientId int, msg Message) {
	log.Printf("Server: SEND to C%d  : %v", clientId, msg.Data)
	s.SendChans[clientId] <- msg
}

// Handles the reception of a given message.
func (s *Server) Handle(msg Message) {
	log.Printf("Server: RECV from C%d: %v", msg.SrcId, msg.Data)

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
func (s *Server) ConnectClient(clientId int, serverToClientChan chan<- Message) {
	s.SendChans[clientId] = serverToClientChan
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

			// Close receiving channel
			close(s.RecvChan)
			return
		}
	}
}
