package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"fmt"
	"time"
	"log"
	"strings"
	"errors"
)


func ComputeNodeEntryTime(n nodetypes.Node, CSDelay time.Duration) time.Duration {
	start := time.Now()
	n.AcquireLock()
	time.Sleep(CSDelay)
	n.ReleaseLock()
	return time.Since(start)
}

func main() {
	nodeCount := 100
	useLogBuf() // Disable logging

	// Lamport's Shared Priority Queue
	fmt.Printf("\n---LAMPORT'S SHARED PRIORITY QUEUE---\n")
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNLamportNodes(nodeCount, sm)
	sysTiming, nodeTiming, err := TimeSystem(o, 100 * time.Millisecond)
	if err != nil {
		fmt.Printf("Lamport's Shared Priority Queue:\n\tSystem Time: %v\n\tError:%v\n", sysTiming, err)
	} else {
		fmt.Printf("Lamport's Shared Priority Queue:\n\tSystem Time: %v\n\tNode Times: %v\n", sysTiming, nodeTiming)
	}
	
	// Ricart and Agrawala
	fmt.Printf("\n---RICART AND AGRAWALA'S OPTIMISATION---\n")
	sm = nodetypes.NewSharedMemory()
	o = NewOrchestratorWithNRicartNodes(nodeCount, sm)
	sysTiming, nodeTiming, err = TimeSystem(o, 100 * time.Millisecond)
	if err != nil {
		fmt.Printf("Ricart and Agrawala's Optimisation:\n\tSystem Time: %v\n\tError:%v\n", sysTiming, err)
	} else {
		fmt.Printf("Ricart and Agrawala's Optimisation:\n\tSystem Time: %v\n\tNode Times: %v\n", sysTiming, nodeTiming)
	}
	
	// Voting Protocol
	fmt.Printf("\n---VOTING PROTOCOL---\n")
	sm = nodetypes.NewSharedMemory()
	o = NewOrchestratorWithNVoterNodes(nodeCount, sm)
	sysTiming, nodeTiming, err = TimeSystem(o, 100 * time.Millisecond)
	if err != nil {
		fmt.Printf("Voting Protocol:\n\tSystem Time: %v\n\tError:%v\n", sysTiming, err)
	} else {
		fmt.Printf("Voting Protocol:\n\tSystem Time: %v\n\tNode Times: %v\n", sysTiming, nodeTiming)
	}
}

// Returns:
// - Time taken for the entire system to complete
// - Time taken for each node to enter and then exit the CS
func TimeSystem(o *Orchestrator, CSDelay time.Duration) (time.Duration, map[int]time.Duration, error) {
	start := time.Now()
	nodeTimings := make(map[int]time.Duration)
	nodeRecordChan := make(chan nodeRecord)
	o.Init()

	for nodeId, node := range o.nodes {
		nodeId := nodeId
		go func(id int, n nodetypes.Node) {
			nodeRecordChan <- nodeRecord{nodeId, ComputeNodeEntryTime(n, CSDelay)}
		}(nodeId, node)
	}

	recordCount := 0
	expectedRecords := len(o.nodes)
	for record := range nodeRecordChan {
		recordCount++
		nodeTimings[record.nodeId] = record.timing
		if recordCount == expectedRecords {
			if err := o.Shutdown(); err != nil {
				return time.Since(start), nil, err
			}
			return time.Since(start), nodeTimings, nil
		}
	}

	return time.Since(start), nil, errors.New("Unexpected error.")
}

type nodeRecord struct {
	nodeId int
	timing time.Duration
}


func NewOrchestratorWithNLamportNodes(nodeCount int, sm *nodetypes.SharedMemory) *Orchestrator {
	nodes := make(map[int](nodetypes.Node))
	nodeIds := make([]int, 0)
	endpoints := make([]nodetypes.LamportNodeEndpoint, 0)
	for nodeId := 0; nodeId < nodeCount; nodeId++ {
		nodeIds = append(nodeIds, nodeId)
		endpoints = append(endpoints, nodetypes.NewLamportNodeEndpoint(nodeId))
	}
	for _, nodeId := range(nodeIds) {
		nodes[nodeId] = nodetypes.NewLamportNode(nodeId, endpoints, sm)
	}

	return NewOrchestrator(nodes, sm)
}

func NewOrchestratorWithNRicartNodes(nodeCount int, sm *nodetypes.SharedMemory) *Orchestrator {
	nodes := make(map[int](nodetypes.Node))
	nodeIds := make([]int, 0)
	endpoints := make([]nodetypes.RicartNodeEndpoint, 0)
	for nodeId := 0; nodeId < nodeCount; nodeId++ {
		nodeIds = append(nodeIds, nodeId)
		endpoints = append(endpoints, nodetypes.NewRicartNodeEndpoint(nodeId))
	}
	for _, nodeId := range(nodeIds) {
		nodes[nodeId] = nodetypes.NewRicartNode(nodeId, endpoints, sm)
	}

	return NewOrchestrator(nodes, sm)
}

func NewOrchestratorWithNVoterNodes(nodeCount int, sm *nodetypes.SharedMemory) *Orchestrator {
	nodes := make(map[int](nodetypes.Node))
	nodeIds := make([]int, 0)
	endpoints := make([]nodetypes.VoterNodeEndpoint, 0)
	for nodeId := 0; nodeId < nodeCount; nodeId++ {
		nodeIds = append(nodeIds, nodeId)
		endpoints = append(endpoints, nodetypes.NewVoterNodeEndpoint(nodeId))
	}
	for _, nodeId := range(nodeIds) {
		nodes[nodeId] = nodetypes.NewVoterNode(nodeId, endpoints, sm)
	}

	return NewOrchestrator(nodes, sm)
}

// Simple struct to contain contents of log
type logBuffer struct {
	contents []string
}

func (tl *logBuffer) Write(p []byte) (int, error) {
	tl.contents = append(tl.contents, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (tl *logBuffer) Dump() {
	for _, line := range tl.contents {
		fmt.Println(line)
	}
}

// Makes log use a temporary log that is returned
func useLogBuf() *logBuffer {
	logBuf := logBuffer{make([]string, 0)}
	log.SetPrefix("")
	log.SetFlags(log.Ltime)
	log.SetOutput(&logBuf)

	return &logBuf
}
