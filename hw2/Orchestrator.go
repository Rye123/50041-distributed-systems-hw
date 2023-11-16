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

func (o *Orchestrator) Init() (err error) {
	for _, n := range o.nodes {
		n.Init()
	}
	return nil
}

func (o *Orchestrator) NodeEnter(nodeId int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()
	log.Printf("N%d: Request to enter CS", nodeId)
	o.nodes[nodeId].AcquireLock()
	log.Printf("N%d: Entered CS", nodeId)
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
