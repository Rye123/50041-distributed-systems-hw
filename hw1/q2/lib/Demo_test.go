package lib

import (
	"fmt"
	"log"
	"testing"
	"time"
)

const DEMO_SEND_INTV = 5 * time.Second
const DEMO_TIMEOUT = 2 * time.Second
const DEMO_NODECOUNT = 10

func systemPrint(s string) {
	fmt.Printf("\n---SYSTEM: %v---\n", s)
}

func Test_BasicInit_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO BasicInit Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running BasicInit with %d nodes.---", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))
}

func Test_BasicSync_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO BasicSync Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running BasicSync with %d nodes.---", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// Update value
	update_value := "Hello there."
	systemPrint(fmt.Sprintf("Updating coordinator N%d with value: \"%v\".", coordId, update_value))
	o.UpdateNodeValue(coordId, update_value, true)
	systemPrint(fmt.Sprintf("Updated. Waiting for (send_interval + RTT/2) for values to propagate."))

	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	// Detect value
	values := o.GetValues()
	systemPrint(fmt.Sprintf("Wait complete. Printing values."))
	for nodeId, value := range values {
		fmt.Printf("SYSTEM: N%d Value : %v\n", nodeId, value)
	}
}

func Test_BasicCrashAndReboot_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO BasicCrashAndReboot Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running BasicCrashAndReboot with %d nodes.", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))

	// Killing of coordinator
	systemPrint(fmt.Sprintf("Killing off coordinator node N%d.", coordId))
	o.KillNode(coordId)
	systemPrint(fmt.Sprintf("N%d killed. Blocking until election is completed...", coordId))
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	newCoordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", newCoordId))

	// Reboot original coordinator
	systemPrint(fmt.Sprintf("Rebooting original coordinator node N%d...", coordId))
	o.RestartNode(coordId)
	systemPrint(fmt.Sprintf("Rebooted N%d. Blocking until election is completed...", coordId))
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", coordId))
}

func Test_NonCoordCrashDuringElection_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO NonCoordCrashDuringElection Test skipped in short mode.")
	}

	if DEMO_NODECOUNT < 3 {
		panic("Please set DEMO_NODECOUNT to be greater than 3 nodes.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running NonCoordCrashDuringElection with %d nodes.", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))

	// Killing of coordinator
	systemPrint(fmt.Sprintf("Killing off coordinator node N%d.", coordId))
	o.KillNode(coordId)
	systemPrint(fmt.Sprintf("N%d killed. Blocking until election starts...", coordId))
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)

	// Killing of arbitrary non-coordinator during election
	arbId := NodeId(coordId - 2)
	o.KillNode(arbId)
	systemPrint(fmt.Sprintf("N%d killed. Blocking until election is over...", arbId))
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	newCoordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", newCoordId))

	// Reboot arbitrary non-coordinator
	systemPrint(fmt.Sprintf("Rebooting non-coordinator node N%d...", arbId))
	o.RestartNode(arbId)
	systemPrint(fmt.Sprintf("Rebooted N%d. Blocking until election is completed...", arbId))
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))
}

func Test_PartialAnnouncement_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO PartialAnnouncement Test skipped in short mode.")
	}

	if DEMO_NODECOUNT < 5 {
		panic("Please set DEMO_NODECOUNT to 5 or more nodes.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running PartialAnnouncement with %d nodes.", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	// Killing of coordinator and second coordinator
	o.KillNode(DEMO_NODECOUNT - 1)
	o.KillNode(DEMO_NODECOUNT - 2)
	systemPrint(fmt.Sprintf("Coordinator and future coordinator killed."))

	// Manually set node coordinators to simulate partial announcement
	halfId := (DEMO_NODECOUNT - 2) / 2
	for i := 0; i < halfId; i++ {
		o.Nodes[NodeId(i)].CoordinatorId = DEMO_NODECOUNT - 2
	}
	systemPrint(fmt.Sprintf("Nodes with IDs from %d to %d have coordinator N%d.", 0, halfId, DEMO_NODECOUNT-1))
	for i := halfId; i < DEMO_NODECOUNT-2; i++ {
		o.Nodes[NodeId(i)].CoordinatorId = DEMO_NODECOUNT - 1
	}
	systemPrint(fmt.Sprintf("Nodes with IDs from %d to %d have coordinator N%d.", halfId, DEMO_NODECOUNT-2, DEMO_NODECOUNT))

	// Initialise the messed-up system
	o.Initiate()
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", coordId))
}

func Test_SyncCrashReboot_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO SyncCrashReboot Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running SyncCrashReboot with %d nodes.", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))

	// Update values
	update_value := "i love the bully algorithm"
	systemPrint(fmt.Sprintf("Updating N%d with value \"%v\".", coordId, update_value))
	o.UpdateNodeValue(coordId, update_value, true)
	systemPrint(fmt.Sprintf("Updated. Waiting for (send_interval + RTT/2) for values to propagate."))

	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	// Detect value
	values := o.GetValues()
	systemPrint(fmt.Sprintf("Wait complete. Printing values."))
	for nodeId, value := range values {
		fmt.Printf("SYSTEM: N%d Value : %v\n", nodeId, value)
	}

	// Kill coordinator node
	systemPrint(fmt.Sprintf("Killing off coordinator node N%d.", coordId))
	o.KillNode(coordId)
	systemPrint(fmt.Sprintf("N%d killed. Blocking until election is completed...", coordId))
	time.Sleep(DEFAULT_TIMEOUT/2 + DEFAULT_SEND_INTV) // Wait for nodes to detect
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)
	newCoordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", newCoordId))

	// Update second coordinator
	update_value = "no i dont"
	systemPrint(fmt.Sprintf("Updating N%d with value \"%v\".", newCoordId, update_value))
	o.UpdateNodeValue(newCoordId, update_value, true)
	systemPrint(fmt.Sprintf("Updated. Waiting for (send_interval + RTT/2) for values to propagate."))

	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	// Reboot original coordinator
	systemPrint(fmt.Sprintf("Rebooting original coordinator node N%d...", coordId))
	o.RestartNode(coordId)
	systemPrint(fmt.Sprintf("Rebooted N%d. Blocking until election is completed...", coordId))
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", coordId))

	// Update values
	update_value = "yes i do"
	systemPrint(fmt.Sprintf("Updating N%d with value \"%v\".", coordId, update_value))
	o.UpdateNodeValue(coordId, update_value, true)
	systemPrint(fmt.Sprintf("Updated. Waiting for (send_interval + RTT/2) for values to propagate."))

	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	// Detect value
	values = o.GetValues()
	systemPrint(fmt.Sprintf("Wait complete. Printing values."))
	for nodeId, value := range values {
		fmt.Printf("SYSTEM: N%d Value : %v\n", nodeId, value)
	}
}

func Test_SimulateBestCase_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO SimulateBestCase Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running SimulateBestCase with %d nodes. (Wait until first election is done to see the re-election)---", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))
	systemPrint("(Simulating best case, by disabling dead coordinator detection for all except second-highest ID node)")

	for nodeId := range o.Nodes {
		if nodeId == coordId || nodeId == NodeId(coordId-1) {
			continue
		}
		o.Nodes[nodeId].DisableDeadCoordDetection()
	}

	systemPrint(fmt.Sprintf("Killing coordinator N%d.", coordId))
	o.KillNode(coordId)
	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", coordId))
}

func Test_SimulateWorstCase_DEMO(t *testing.T) {
	if testing.Short() {
		t.Skip("DEMO SimulateWorstCase Test skipped in short mode.")
	}

	// Initialise with DEMO_NODECOUNT nodes
	systemPrint(fmt.Sprintf("Running SimulateWorstCase with %d nodes. (Wait until first election is done to see the re-election)---", DEMO_NODECOUNT))
	o := NewOrchestrator(DEMO_NODECOUNT, DEMO_SEND_INTV, DEMO_TIMEOUT)
	defer o.Exit()

	o.Initiate()
	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err := o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Election complete with coordinator N%d.", coordId))
	systemPrint("(Simulating worst case, by disabling dead coordinator detection for all except lowest ID node)")

	for nodeId := range o.Nodes {
		if nodeId == coordId || nodeId == 0 {
			continue
		}
		o.Nodes[nodeId].DisableDeadCoordDetection()
	}

	systemPrint(fmt.Sprintf("Killing coordinator N%d.", coordId))
	o.KillNode(coordId)
	time.Sleep(DEMO_SEND_INTV + DEMO_TIMEOUT/2) // Wait for the maximum time for values to be propagated (Send interval + RTT/2)

	o.BlockTillElectionStart(5, time.Second)
	o.BlockTillElectionDone(5, time.Second)

	// Detect coordinator
	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	coordId, err = o.BlockUntilCoordinatorConsistent(10, time.Second)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	systemPrint(fmt.Sprintf("Re-election complete with coordinator N%d.", coordId))
}
