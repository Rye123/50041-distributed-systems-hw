package lib

import (
	"log"
	"strings"
	"testing"
	"time"
)

const DEFAULT_SEND_INTV = 5 * time.Second
const DEFAULT_TIMEOUT = 1 * time.Second

// Simple struct to contain contents of log
type tempLog struct {
	contents []string
}

func (tl *tempLog) Write(p []byte) (int, error) {
	tl.contents = append(tl.contents, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (tl *tempLog) Dump(t *testing.T) {
	for _, line := range(tl.contents) {
		t.Log(line)
	}
}

// Makes log use a temporary log that is returned
func useTempLog() *tempLog {
	tempLog := tempLog{make([]string, 0)}
	log.SetPrefix("")
	log.SetFlags(log.Ltime)
	log.SetOutput(&tempLog)

	return &tempLog
}

/**
  --- TEST INITIALISATION ---
  These test the basic implementation of the Bully Algorithm.

  My implementation has ALL nodes start elections when they join.
  Hence, these simulate the case where multiple nodes start their own elections simultaneously.
*/


// Upon initialisation, should eventually end up with a final coordinator ID.
func TestInitWith1Node(t *testing.T) {
	// Test with 1 node
	tLog := useTempLog()
	o := NewOrchestrator(1, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("1-node test failed: %v", err)
	} else {
		if coordId != 0 {
			tLog.Dump(t)
			t.Fatalf("1-node test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}

// Upon initialisation, should eventually end up with a final coordinator ID.
func TestInitWith5Nodes(t *testing.T) {
	// Test with 5 nodes
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node test failed: %v", err)
	} else {
		if coordId != 4 {
			tLog.Dump(t)
			t.Fatalf("5-node test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}

// Upon initialisation, should eventually end up with a final coordinator ID.
func TestInitWith10Nodes(t *testing.T) {
	// Test with 10 nodes
	tLog := useTempLog()
	o := NewOrchestrator(10, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("10-node test failed: %v", err)
	} else {
		if coordId != 9 {
			tLog.Dump(t)
			t.Fatalf("10-node test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}


// Upon initialisation, should eventually end up with a final coordinator ID.
func TestInitWith50Nodes(t *testing.T) {
	// Test with 50 nodes
	tLog := useTempLog()
	o := NewOrchestrator(50, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("50-node test failed: %v", err)
	} else {
		if coordId != 49 {
			tLog.Dump(t)
			t.Fatalf("50-node test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}


/**
  TEST COORDINATOR CRASH
  These test cases tests what happens when the coordinator goes down.
*/


// After initialisation, coordinator crash should result in new coordinator
func TestCoordCrashWith5Nodes(t *testing.T) {
	// Initialisation: Might as well test this as well :)
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node test failed: %v", err)
	} else {
		if coordId != 4 {
			tLog.Dump(t)
			t.Fatalf("5-node test failed: coordinator is N%d", coordId)
		}
	}

	// Killing of coordinator
	o.KillNode(4)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node coordinator crash test failed: %v", err)
	} else {
		if coordId != 3 {
			tLog.Dump(t)
			t.Fatalf("5-node coordinator crash test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}

// After initialisation, coordinator crash should result in new coordinator
func TestCoordCrashWith25Nodes(t *testing.T) {
	// Initialisation: Might as well test this as well :)
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("25-node test failed: %v", err)
	} else {
		if coordId != 24 {
			tLog.Dump(t)
			t.Fatalf("25-node test failed: coordinator is N%d", coordId)
		}
	}

	// Killing of coordinator
	o.KillNode(24)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("25-node coordinator crash test failed: %v", err)
	} else {
		if coordId != 23 {
			tLog.Dump(t)
			t.Fatalf("25-node coordinator crash test failed: coordinator is N%d", coordId)
		}
	}

	o.Exit()
}



// After initialisation, coordinator crash should result in new coordinator, and reset to original coordinator upon reboot
func TestCoordCrashAndRebootWith5Nodes(t *testing.T) {
	// Initialisation: Might as well test this as well :)
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node test failed: %v", err)
	} else {
		if coordId != 4 {
			tLog.Dump(t)
			t.Fatalf("5-node test failed: coordinator is N%d", coordId)
		}
	}

	// Killing of coordinator
	o.KillNode(4)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node coordinator crash test failed: %v", err)
	} else {
		if coordId != 3 {
			tLog.Dump(t)
			t.Fatalf("5-node coordinator crash test failed: coordinator is N%d", coordId)
		}
	}

	// Reboot of original coordinator
	o.RestartNode(4)
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("5-node reboot test failed: %v", err)
	} else {
		if coordId != 4 {
			tLog.Dump(t)
			t.Fatalf("5-node reboot test failed: coordinator is N%d", coordId)
		}
	}
	

	o.Exit()
}


// After initialisation, coordinator crash should result in new coordinator, and reset to original coordinator upon reboot
func TestCoordCrashAndRebootWith25Nodes(t *testing.T) {
	// Initialisation: Might as well test this as well :)
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err := o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("25-node test failed: %v", err)
	} else {
		if coordId != 24 {
			tLog.Dump(t)
			t.Fatalf("25-node test failed: coordinator is N%d", coordId)
		}
	}

	// Killing of coordinator
	o.KillNode(24)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("25-node coordinator crash test failed: %v", err)
	} else {
		if coordId != 23 {
			tLog.Dump(t)
			t.Fatalf("25-node coordinator crash test failed: coordinator is N%d", coordId)
		}
	}

	// Reboot of original coordinator
	o.RestartNode(24)
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	coordId, err = o.GetCoordinatorId(10, time.Second)

	if err != nil {
		tLog.Dump(t)
		t.Fatalf("25-node reboot test failed: %v", err)
	} else {
		if coordId != 24 {
			tLog.Dump(t)
			t.Fatalf("25-node reboot test failed: coordinator is N%d", coordId)
		}
	}
	

	o.Exit()
}
