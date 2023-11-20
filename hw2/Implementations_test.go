/**
  IMPLEMENTATIONS_TEST
  System tests for all three protocols. This doesn't print logs unless the test fails.

  We test using nodetypes.SharedMemory. This is a **non-threadsafe** piece of memory used to represent an unsafe critical section, where users of the critical section must implement their own functions for mutual exclusion.
  - The EnterCS panics the moment a node calls it when another node is still in the critical section.
  - This allows as an efficient way of detecting safety violations in our implementations.

  In general, here we use a channel to detect when the orchestrator functions are completed, and call Shutdown() when this is done.
*/

package main

import (
	"testing"
	"time"
	"1005129_RYAN_TOH/hw2/nodetypes"
)

// How many nodes to use
const TEST_NODE_COUNT = 100

// How long a node stays in the critical section
const TEST_CS_DELAY = 10 * time.Millisecond

func TestStandard_Lamport(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNLamportNodes(TEST_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error, (TEST_NODE_COUNT + 10) * 2)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := TEST_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestHalfConcurrent_Lamport(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNLamportNodes(TEST_NODE_COUNT, sm)
	o.Init()

	nodesToEnter := TEST_NODE_COUNT/2
	errChan := make(chan error, (nodesToEnter + 10) * 2)

	count := 0
	for nodeId := range o.nodes {
		if count == nodesToEnter {
			break
		}
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
		count++
	}

	routineCount := nodesToEnter * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestStandard_Ricart(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNRicartNodes(TEST_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := TEST_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestHalfConcurrent_Ricart(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNRicartNodes(TEST_NODE_COUNT, sm)
	o.Init()

	nodesToEnter := TEST_NODE_COUNT/2
	errChan := make(chan error, (nodesToEnter + 10) * 2)

	count := 0
	for nodeId := range o.nodes {
		if count == nodesToEnter {
			break
		}
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
		count++
	}

	routineCount := nodesToEnter * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

func TestStandard_Voting(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNVoterNodes(TEST_NODE_COUNT, sm)
	o.Init()

	errChan := make(chan error, (TEST_NODE_COUNT + 10) * 2)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := TEST_NODE_COUNT * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}


func TestHalfConcurrent_Voting(t *testing.T) {
	tLog := useTempLog(t)
	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNVoterNodes(TEST_NODE_COUNT, sm)
	o.Init()

	nodesToEnter := TEST_NODE_COUNT/2
	errChan := make(chan error, (nodesToEnter + 10) * 2)

	count := 0
	for nodeId := range o.nodes {
		if count == nodesToEnter {
			break
		}
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(TEST_CS_DELAY)
			errChan <- o.NodeExit(id)
		}(nodeId)
		count++
	}

	routineCount := nodesToEnter * 2
	curCount := 0
	for err := range errChan {
		curCount++
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		if curCount == routineCount {
			shutdownErr := o.Shutdown()
			if shutdownErr != nil {
				tLog.Dump()
				t.Fatalf("ERROR: %v", shutdownErr)
			}
			break
		}
	}
}

