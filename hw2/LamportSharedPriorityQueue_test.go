package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"testing"
	"time"
)

/*
   Our system is validated by the use of nodetypes.SharedMemory.

   This struct provides the functions EnterCS and ExitCS, and panics the moment multiple nodes are IN the section -- the responsibility of restricting entry is on the implementation of the nodes that call EnterCS and ExitCS.
*/

func TestStandardUsageLamport(t *testing.T) {
	//tLog := useTempLog(t)
	nodeCount := 100

	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNLamportNodes(nodeCount, sm)
	o.Init()
	errChan := make(chan error)

	for nodeId := range o.nodes {
		nodeId := nodeId
		go func(id int) {
			errChan <- o.NodeEnter(id)
			time.Sleep(10 * time.Millisecond)
			errChan <- o.NodeExit(id)
		}(nodeId)
	}

	routineCount := nodeCount * 2
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
	//tLog.Dump()
}
