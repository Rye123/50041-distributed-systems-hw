package nodetypes

import (
	"fmt"
)

type ClockVal int
func MaxClockVal(c1, c2 ClockVal) ClockVal {
	if c1 > c2 {
		return c1
	}
	return c2
}

// Record of who obtained the lock at what timestamp
type smData struct {
	lockHolder int
	timestamp ClockVal
}

type Node interface {
	Init() error
	AcquireLock()
	ReleaseLock()
	Shutdown() error
}

type SharedMemory struct {
	history []smData
	currentHolderId int // current holder, default is -1. this is the shared memory that is being modified and checked
}

func NewSharedMemory() *SharedMemory {
	return &SharedMemory{make([]smData, 0), -1}
}

func (sm *SharedMemory) DumpHistory() string {
	ret := "["
	for _, smDat := range sm.history {
		ret += fmt.Sprintf("N%d: %d, ", smDat.lockHolder, smDat.timestamp)
	}

	ret = ret[:len(ret)-2] + "]"

	return ret
}

// Enter the critical section.
func (sm *SharedMemory) EnterCS(nodeId int, timestamp ClockVal) {
	if sm.currentHolderId == nodeId {
		panic(fmt.Sprintf("Node %d tried to start CS again", nodeId))
	}
	sm.history = append(sm.history, smData{nodeId, timestamp})

	if sm.currentHolderId != -1 {
		panic(fmt.Sprintf("Safety condition breached: Current holder is %d, but node %d entered too.", sm.currentHolderId, nodeId))
	}

	sm.currentHolderId = nodeId
}

// Exits the critical section
func (sm *SharedMemory) ExitCS(nodeId int) {
	if sm.currentHolderId != nodeId {
		if sm.currentHolderId == -1 {
			panic(fmt.Sprintf("Node %d tried to exit CS without entering.", nodeId))
		}
		panic(fmt.Sprintf("Node %d tried to exit CS, but node %d is inside.", nodeId, sm.currentHolderId))
	}

	sm.currentHolderId = -1
	
}
