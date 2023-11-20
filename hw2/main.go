package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type sysTimingRecord struct {
	nodesEnteringCS int
	timing time.Duration
	err error
}

func main() {
	nodeCount := 10
	csDelay := 100 * time.Millisecond
	useLogBuf() // Disable logging

	// Lamport's Shared Priority Queue
	fmt.Printf("\n---LAMPORT'S SHARED PRIORITY QUEUE---\n")
	fmt.Printf("Progress: ")
	sysTimes := make(map[int]time.Duration)
	sysTimesChan := make(chan sysTimingRecord, nodeCount + 10)
	//// Initialise goroutines
	for i := 1; i <= nodeCount; i++ {
		i := i
		sm := nodetypes.NewSharedMemory()
		o := NewOrchestratorWithNLamportNodes(nodeCount, sm)
		record := TimeSystem(o, csDelay, i)
		if record.err != nil {
			fmt.Printf("Error with %d nodes entering CS: %v\n", record.nodesEnteringCS, record.err)
		}
		fmt.Printf("|")
		sysTimes[record.nodesEnteringCS] = record.timing
	}
	fmt.Println()
	//// Provide output
	fmt.Printf("NODES ENTERING | TIME TAKEN\n")
	for i := 1; i <= nodeCount; i++ {
		fmt.Printf("%14d | %s\n", i, sysTimes[i].Round(time.Millisecond).String())
	}
	close(sysTimesChan)
	
	
	// Ricart and Agrawala
	fmt.Printf("\n---RICART AND AGRAWALA'S OPTIMISATION---\n")
	fmt.Printf("Progress: ")
	sysTimes = make(map[int]time.Duration)
	sysTimesChan = make(chan sysTimingRecord, nodeCount + 10)
	//// Initialise goroutines
	for i := 1; i <= nodeCount; i++ {
		i := i
		sm := nodetypes.NewSharedMemory()
		o := NewOrchestratorWithNRicartNodes(nodeCount, sm)
		record := TimeSystem(o, csDelay, i)
		if record.err != nil {
			fmt.Printf("Error with %d nodes entering CS: %v\n", record.nodesEnteringCS, record.err)
		}
		fmt.Printf("|")
		sysTimes[record.nodesEnteringCS] = record.timing
	}
	fmt.Println()
	//// Provide output
	fmt.Printf("NODES ENTERING | TIME TAKEN\n")
	for i := 1; i <= nodeCount; i++ {
		fmt.Printf("%14d | %s\n", i, sysTimes[i].Round(time.Millisecond).String())
	}
	close(sysTimesChan)
	
	
	// Voting Protocol
	fmt.Printf("\n---VOTING PROTOCOL---\n")
	fmt.Printf("Progress: ")
	sysTimes = make(map[int]time.Duration)
	sysTimesChan = make(chan sysTimingRecord, nodeCount + 10)
	//// Initialise goroutines
	for i := 1; i <= nodeCount; i++ {
		i := i
		sm := nodetypes.NewSharedMemory()
		o := NewOrchestratorWithNVoterNodes(nodeCount, sm)
		record := TimeSystem(o, csDelay, i)
		if record.err != nil {
			fmt.Printf("Error with %d nodes entering CS: %v\n", record.nodesEnteringCS, record.err)
		}
		fmt.Printf("|")
		sysTimes[record.nodesEnteringCS] = record.timing
	}
	fmt.Println()
	//// Provide output
	fmt.Printf("NODES ENTERING | TIME TAKEN\n")
	for i := 1; i <= nodeCount; i++ {
		fmt.Printf("%14d | %s\n", i, sysTimes[i].Round(time.Millisecond).String())
	}
	close(sysTimesChan)
	
}

// Returns time taken for the entire system, when `nodesToEnter` nodes enter the CS simultaneously.
func TimeSystem(o *Orchestrator, CSDelay time.Duration, nodesToEnter int) sysTimingRecord {
	errChan := make(chan error)
	retErr := error(nil)
	start := time.Now()
	o.Init()

	// Make `nodesToEnter` nodes enter simultaneously
	nodesEntered := 0
	for nodeId := range o.nodes {
		if nodesEntered == nodesToEnter {
			break
		}

		id := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(CSDelay)
			errChan <- o.NodeExit(id)
		}(id)
		nodesEntered++
	}

	// Wait for all goroutines to complete
	curCount := 0
	for {
		select {
		case err := <-errChan:
			curCount++
			if err != nil {
				retErr = err
			}

			if curCount == nodesEntered * 2 {
				if shutdownErr := o.Shutdown(); shutdownErr != nil {
					retErr = err
				}
				return sysTimingRecord{nodesToEnter, time.Since(start), retErr}
			} else if curCount > nodesEntered * 2 {
				panic(fmt.Sprintf("Unexpected curCount %d at nodes %d", curCount, nodesToEnter))
			}
		case <-time.After(120 * time.Second):
			return sysTimingRecord{nodesToEnter, time.Since(start), errors.New("Timeout after 120 seconds.")}
		}

		
	}
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
