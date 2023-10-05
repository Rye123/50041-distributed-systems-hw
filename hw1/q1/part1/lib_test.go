/*
* Testing package for lib
 */
package part1

import (
	"log"
	"strings"
	"testing"
	"time"
)

// Simple struct to contain the contents of the log for testing
type TempLog struct {
	contents []string
}

func (tempLog *TempLog) Write(p []byte) (int, error) {
	tempLog.contents = append(tempLog.contents, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (tempLog *TempLog) Len() int {
	return len(tempLog.contents)
}

func (tempLog *TempLog) Dump() string {
	ret := ""
	for _, content := range(tempLog.contents) {
		ret += content + "\n"
	}
	return ret
}

// Client send should result in server receive
func TestSendReceiveOnServer(t *testing.T) {
	tempLog := TempLog{make([]string, 0)}
	log.SetPrefix("")
	log.SetFlags(log.Ltime)
	log.SetOutput(&tempLog)

	quit := make(chan bool)

	serverRecvChan := make(chan Message)
	server := NewServer(serverRecvChan, 0, quit)
	client0RecvChan := make(chan Message)
	client0SendChan := make(chan Message)
	client0 := NewClient(0, client0RecvChan, serverRecvChan, (time.Millisecond * 100))
	server.ConnectClient(0, client0RecvChan, client0SendChan)

	// Start Server and Client
	go server.Run()
	go client0.Run()

	// Block until tempLog is at 10 messages and above
	for {
		if tempLog.Len() >= 10 {
			quit <- true
			break
		}
	}

	// Loop through list. If we see a "SERVER | RECV ... | (message)", we expect an earlier "Client ... | SEND to SERVER | (message)"  
	for i, content := range(tempLog.contents) {
		if strings.Contains(content, "SERVER | RECV") {
			senderIndex := -1
			
			// Extract the message
			msg := strings.Split(content, " | ")[2]

			// Find the earlier message
			for j, content2 := range(tempLog.contents) {
				if i == j {
					continue
				}
				if strings.Contains(content2, "SEND to SERVER") {
					// Extract the message
					msg2 := strings.Split(content2, " | ")[2]
					if msg == msg2 {
						senderIndex = j
						break
					}
				}
			}

			if senderIndex == -1 {
				t.Fatalf("Message \"%v\" was not sent. Full dump:\n%v", msg, tempLog.Dump())
			} else if senderIndex > i {
				t.Fatalf("Message \"%v\" was received BEFORE send. Full dump:\n%v", msg, tempLog.Dump())
			}
			
		}
	}

	
}
