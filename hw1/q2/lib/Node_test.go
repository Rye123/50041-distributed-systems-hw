package lib

import (
	"log"
	"strings"
	"testing"
	"time"
)

const DEFAULT_SEND_INTV = 5 * time.Second
const DEFAULT_TIMEOUT = 2 * time.Second

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

/** HELPER FUNCTIONS */

// Throws a fatal error if the coordinator ID doesn't match the expected ID.
func assertCoordinatorId(t *testing.T, o *Orchestrator, tLog *tempLog, expectedId NodeId) {
	coordId, err := o.GetCoordinatorId(10, time.Second)
	if err != nil {
		tLog.Dump(t)
		t.Fatalf("Test failed: %v", err)
	} else {
		if coordId != expectedId {
			tLog.Dump(t)
			t.Fatalf("Test failed: Coordinator is %d, expected %d", coordId, expectedId)
		}
	}
}

// Throws a fatal error if the value stored in the system doesn't match the expected value.
func assertOverallValue(t *testing.T, o *Orchestrator, tLog *tempLog, expectedValue string) {
	value, err := o.GetValue()
	if err != nil {
		tLog.Dump(t)
		t.Fatalf("Test failed: %v", err)
	} else {
		if value != expectedValue {
			tLog.Dump(t)
			t.Fatalf("Test failed: Value is %s, expected %s", value, expectedValue)
		}
	}
}

/**
  --- BASIC INITIALISATION ---
  These test the basic implementation of the Bully Algorithm.
  1. Start up N nodes, wait for election to complete.
  2. Ensure coordinator node is the one with the highest ID (i.e. ID (N-1)).
*/

func Test_BasicInit_1Node(t *testing.T) {
	// Test with 1 node
	tLog := useTempLog()
	o := NewOrchestrator(1, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	assertCoordinatorId(t, o, tLog, 0)

	o.Exit()
}

func Test_BasicInit_5Nodes(t *testing.T) {
	// Test with 5 nodes
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 4)

	o.Exit()
}

func Test_BasicInit_10Nodes(t *testing.T) {
	// Test with 10 nodes
	tLog := useTempLog()
	o := NewOrchestrator(10, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 9)

	o.Exit()
}

func Test_BasicInit_50Nodes(t *testing.T) {
	// Test with 50 nodes
	tLog := useTempLog()
	o := NewOrchestrator(50, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	assertCoordinatorId(t, o, tLog, 49)

	o.Exit()
}


/**
  ---COMPLEX INITIALISATION---
  Tests the scenario where somehow we have multiple nodes considering themselves coordinators.
  This is possible if the veto timeout (i.e. the time a node waits for a veto)
  is too short, and the node considers itself a coordinator as a result.

  This should be resolved by the real coordinator detecting a broadcast from lower ID 'coordinators',
  and automatically resolving that by sending the lower ID coordinator an announcement message.
*/
func Test_ComplexInit(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	assertCoordinatorId(t, o, tLog, 24)

	// Here, we modify the coordinator IDs.
	tLog = useTempLog()  // Clear previous tempLog
	log.Println("Orchestrator: Manually modifying coordinator IDs.")
	o.Nodes[0].CoordinatorId = o.Nodes[0].Id
	o.Nodes[4].CoordinatorId = o.Nodes[4].Id
	o.Nodes[7].CoordinatorId = o.Nodes[7].Id
	o.Nodes[13].CoordinatorId = o.Nodes[13].Id
	o.Nodes[19].CoordinatorId = o.Nodes[19].Id
	o.Nodes[20].CoordinatorId = o.Nodes[20].Id
	o.Nodes[22].CoordinatorId = o.Nodes[22].Id
	o.Nodes[23].CoordinatorId = o.Nodes[23].Id
	
	coordId, err := o.GetCoordinatorId(0, time.Second)
	if err == nil {
		tLog.Dump(t)
		t.Fatalf("Failed to mess up the coordinators, coordinator ID is %d", coordId)
	}

	time.Sleep(DEFAULT_SEND_INTV + DEFAULT_TIMEOUT) // Max time for propagation of messages

	// Coordinator ID should be resolved.
	assertCoordinatorId(t, o, tLog, 24)
}


/**
  ---BASIC SYNCHRONISATION---
  Tests synchronisation of values among nodes.
  1. Start up N nodes, wait for election to complete.
  2. Update coordinator with new value, wait for (RTT + default timeout) for the value to be propagated.
  3. Ensure ALL nodes have the same modified value.
*/

func Test_BasicSync_5Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 4)

	// Update values
	o.UpdateNodeValue(4, "testing", true)
	time.Sleep(DEFAULT_SEND_INTV + DEFAULT_TIMEOUT/2) // Send Interval + RTT/2 is max time for propagation

	// Check values
	assertOverallValue(t, o, tLog, "testing")

	o.Exit()
}

func Test_BasicSync_25Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 24)

	// Update values
	o.UpdateNodeValue(24, "testing", true)
	time.Sleep(DEFAULT_SEND_INTV + DEFAULT_TIMEOUT/2) // Send Interval + RTT/2 is max time for propagation

	// Check values
	assertOverallValue(t, o, tLog, "testing")

	o.Exit()
}

// Here, we modify some of the other values and ensure the true value is propagated.
func Test_BasicSync_25Nodes_Corrupted(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 24)

	// Update values
	o.UpdateNodeValue(5, "test", true)
	o.UpdateNodeValue(8, "testi", true)
	o.UpdateNodeValue(12, "TEST", true)
	o.UpdateNodeValue(15, "asdf", true)
	o.UpdateNodeValue(20, "abcde", true)
	o.UpdateNodeValue(24, "testing", true)
	time.Sleep(DEFAULT_SEND_INTV + DEFAULT_TIMEOUT/2) // Send Interval + RTT/2 is max time for propagation

	// Check values
	assertOverallValue(t, o, tLog, "testing")

	o.Exit()
}


/**
  ---BASIC CRASH---
  These test cases tests what happens when the coordinator goes down.

  A coordinator crash should result in a new coordinator being elected.
*/

func Test_BasicCrash_5Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 4)
	tLog = useTempLog() // Clear log before killing the node

	// Killing of coordinator
	o.KillNode(4)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 3)

	o.Exit()
}

func Test_BasicCrash_25Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 24)
	tLog = useTempLog() // Clear log before killing the node

	// Killing of coordinator
	o.KillNode(24)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 23)

	o.Exit()
}


/**
  ---BASIC CRASH AND REBOOT---
  After the crash, system should elect a new coordinator. After reboot of original coordinator, system should elect the highest ID.
  
  1. Start up N nodes, wait for election to complete.
  2. Kill coordinator node.
  3. Ensure node with the next highest ID is selected.
  4. Reboot coordinator node.
  5. Ensure original coordinator (with highest ID) is re-elected.
*/

func Test_BasicCrashAndReboot_5Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(5, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 4)
	tLog = useTempLog() // Clear log before killing the node

	// Killing of coordinator
	o.KillNode(4)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 3)

	// Reboot of original coordinator
	o.RestartNode(4)
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 4)

	o.Exit()
}

// After initialisation, coordinator crash should result in new coordinator, and reset to original coordinator upon reboot
func Test_BasicCrashAndReboot_25Nodes(t *testing.T) {
	// Initialisation
	tLog := useTempLog()
	o := NewOrchestrator(25, DEFAULT_SEND_INTV, DEFAULT_TIMEOUT)
	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 24)
	tLog = useTempLog() // Clear log before killing the node

	// Killing of coordinator
	o.KillNode(24)
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 23)

	// Reboot of original coordinator
	o.RestartNode(24)
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	
	assertCoordinatorId(t, o, tLog, 24)
	

	o.Exit()
}
