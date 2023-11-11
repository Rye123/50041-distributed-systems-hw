package main

import (
	"1005129_RYAN_TOH/hw2/nodetypes"
	"log"
	"fmt"
	"errors"
)


// Data structure that manages the nodes.
type Orchestrator struct {
	nodes map[int](nodetypes.Node)
	sm *nodetypes.SharedMemory
}

func NewOrchestrator(nodes map[int](nodetypes.Node), sm *nodetypes.SharedMemory) *Orchestrator {
	return &Orchestrator{
		nodes,
		sm,
	}
}

func (o *Orchestrator) NodeEnter(nodeId int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()
	log.Printf("N%d: Enter CS", nodeId)
	o.nodes[nodeId].AcquireLock()
	return nil
}

func (o *Orchestrator) NodeExit(nodeId int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()
	log.Printf("N%d: Exit CS", nodeId)
	o.nodes[nodeId].ReleaseLock()
	return nil
}
