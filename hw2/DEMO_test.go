/**
  DEMO_TESTS
  Demonstrate that the systems work.

  We test using nodetypes.SharedMemory. This is a **non-threadsafe** piece of memory used to represent an unsafe critical section, where users of the critical section must implement their own functions for mutual exclusion.
  - The EnterCS panics the moment a node calls it when another node is still in the critical section.
  - This allows as an efficient way of detecting safety violations in our implementations.

  In general, here we use a channel to detect when the orchestrator functions are completed, and call Shutdown() when this is done.
*/

package main

import (
	"testing"
	"time"
	"fmt"
	"1005129_RYAN_TOH/hw2/nodetypes"
)

// How many nodes to use
const DEMO_NODE_COUNT = 100

// How long a node stays in the critical section
const DEMO_CS_DELAY = 10 * time.Millisecond

func TestLamport_DEMO(t *testing.T) {
	if testing.Short() {t.Skip("TestLamport_DEMO skipped (Short Mode).")}
	fmt.Printf("TestLamport_DEMO initialised with %d nodes and CS delay %v.\n", DEMO_NODE_COUNT, DEMO_CS_DELAY)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNLamportNodes(DEMO_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error, (DEMO_NODE_COUNT + 10) * 2)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(DEMO_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := DEMO_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestRicart_DEMO(t *testing.T) {
	if testing.Short() {t.Skip("TestRicart_DEMO skipped (Short Mode).")}
	fmt.Printf("TestRicart_DEMO initialised with %d nodes and CS delay %v.\n", DEMO_NODE_COUNT, DEMO_CS_DELAY)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNRicartNodes(DEMO_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error, (DEMO_NODE_COUNT + 10) * 2)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(DEMO_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := DEMO_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestVoting_DEMO(t *testing.T) {
	if testing.Short() {t.Skip("TestVoting_DEMO skipped (Short Mode).")}
	fmt.Printf("TestVoting_DEMO initialised with %d nodes and CS delay %v.\n", DEMO_NODE_COUNT, DEMO_CS_DELAY)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNVoterNodes(DEMO_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error, (DEMO_NODE_COUNT + 10) * 2)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(DEMO_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := DEMO_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}


