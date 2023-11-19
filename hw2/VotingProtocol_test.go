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

func TestStandardUsageVoting(t *testing.T) {
	//tLog := useTempLog(t)
	nodeCount := 100

	sm := nodetypes.NewSharedMemory()
	o := NewOrchestratorWithNVoterNodes(nodeCount, sm)
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
