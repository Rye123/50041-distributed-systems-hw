package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"testing"
	"log"
	"strings"
)

// Simple struct to contain contents of log
type tempLog struct {
	t *testing.T
	contents []string
}

func (tl *tempLog) Write(p []byte) (int, error) {
	tl.contents = append(tl.contents, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (tl *tempLog) Dump() {
	for _, line := range tl.contents {
		tl.t.Log(line)
	}
}

// Makes log use a temporary log that is returned
func useTempLog(t *testing.T) *tempLog {
	tempLog := tempLog{t, make([]string, 0)}
	log.SetPrefix("")
	log.SetFlags(log.Ltime)
	log.SetOutput(&tempLog)

	return &tempLog
}

func assertNoError(err error, tLog *tempLog) {
	if err != nil {
		log.Printf("Error: %v", err)
		tLog.Dump()
	}
}

func assertError(err error, tLog *tempLog) {
	if err == nil {
		log.Printf("Error: No error detected, expected error")
		tLog.Dump()
	}
}

// Ensure Orchestrator works, by using a Naive implementation to test
func TestOrchestratorNoError(t *testing.T) {
	tLog := useTempLog(t)
	
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodeLs := []int{0, 1, 2, 3}
	nodes[0] = nodetypes.NewNaiveNode(0, nodeLs, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, nodeLs, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, nodeLs, sm)
	nodes[3] = nodetypes.NewNaiveNode(3, nodeLs, sm)

	o := NewOrchestrator(nodes, sm)
	o.Init()
	err := o.NodeEnter(0); assertNoError(err, tLog)
	err = o.NodeExit(0); assertNoError(err, tLog)
	err = o.NodeEnter(1); assertNoError(err, tLog)
	err = o.NodeExit(1); assertNoError(err, tLog)
}

// Expect error when we have the same node enter twice
func TestOrchestratorSameNodeEnterTwice(t *testing.T) {
	tLog := useTempLog(t)
	
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodeLs := []int{0, 1, 2, 3}
	nodes[0] = nodetypes.NewNaiveNode(0, nodeLs, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, nodeLs, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, nodeLs, sm)
	nodes[3] = nodetypes.NewNaiveNode(3, nodeLs, sm)
	o := NewOrchestrator(nodes, sm)
	o.Init()
	err := o.NodeEnter(1); assertNoError(err, tLog)
	err = o.NodeEnter(1); assertError(err, tLog)
}

// Expect error when we have the two nodes enter
func TestOrchestratorTwoNodesEnterCS(t *testing.T) {
	tLog := useTempLog(t)
	
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodeLs := []int{0, 1, 2, 3}
	nodes[0] = nodetypes.NewNaiveNode(0, nodeLs, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, nodeLs, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, nodeLs, sm)
	nodes[3] = nodetypes.NewNaiveNode(3, nodeLs, sm)
	o := NewOrchestrator(nodes, sm)
	o.Init()
	err := o.NodeEnter(1); assertNoError(err, tLog)
}

// Expect error when a node that hasn't entered exits
func TestOrchestratorRandomNodeExit(t *testing.T) {
	tLog := useTempLog(t)
	
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodeLs := []int{0, 1, 2, 3}
	nodes[0] = nodetypes.NewNaiveNode(0, nodeLs, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, nodeLs, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, nodeLs, sm)
	nodes[3] = nodetypes.NewNaiveNode(3, nodeLs, sm)
	o := NewOrchestrator(nodes, sm)
	o.Init()
	err := o.NodeExit(2); assertError(err, tLog)
}

// Expect error when a node that hasn't entered exits with another node inside
func TestOrchestratorNodeEnterAnotherExit(t *testing.T) {
	tLog := useTempLog(t)
	
	sm := nodetypes.NewSharedMemory()
	nodes := make(map[int](nodetypes.Node))
	nodeLs := []int{0, 1, 2, 3}
	nodes[0] = nodetypes.NewNaiveNode(0, nodeLs, sm)
	nodes[1] = nodetypes.NewNaiveNode(1, nodeLs, sm)
	nodes[2] = nodetypes.NewNaiveNode(2, nodeLs, sm)
	nodes[3] = nodetypes.NewNaiveNode(3, nodeLs, sm)
	o := NewOrchestrator(nodes, sm)
	o.Init()
	err := o.NodeEnter(1); assertNoError(err, tLog)
	err = o.NodeExit(2); assertError(err, tLog)
}
