package main

import (
	"log"
	"testing"
	"time"
)

func TestSingleFaultBeforeWrite(t *testing.T) {
	// Tests scenario where primary goes down BEFORE any request occurs.
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()

	sys.KillCM(-1) // CM-2 is left, should take over

	// Write to a node, read from another
	expectedData := "hello"
	sys.NodeWrite(5, 0, expectedData)
	data, consistent := testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	expectedData = "boo"
	expectedData2 := "asdf"
	sys.NodeWrite(10, 0, expectedData)
	sys.NodeWrite(10, 1, expectedData2)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 2, 0, data, expectedData)
	}
	data, consistent = testDataConsistent(sys, 1)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData2 {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 3,1, data, expectedData2)
	}

	sys.Exit()
}

func TestSingleFaultAfterRequest(t *testing.T) {
	// Tests scenario where primary goes down AFTER a request has been completed
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()

	// Write to a node, read from another
	expectedData := "hello"
	sys.NodeWrite(5, 0, expectedData)
	data, consistent := testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.KillCM(-1) // CM-2 is left, should take over

	// Write to a node, read from another
	expectedData = "goodbye"
	sys.NodeWrite(5, 0, expectedData)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	expectedData = "boo"
	expectedData2 := "asdf"
	sys.NodeWrite(10, 0, expectedData)
	sys.NodeWrite(10, 1, expectedData2)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 2, 0, data, expectedData)
	}
	data, consistent = testDataConsistent(sys, 1)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData2 {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 3,1, data, expectedData2)
	}

	sys.Exit()
}


func TestSingleFaultAndReboot(t *testing.T) {
	// Tests scenario where primary fails after election, and reboots, and the new primary fails.
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()

	// Write to a node, read from another
	expectedData := "hello"
	sys.NodeWrite(5, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent := testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.KillCM(-1) // CM-2 is left, should take over

	// Write to a node, read from another
	expectedData = "goodbye"
	sys.NodeWrite(10, 0, expectedData)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.RebootCM(-1) // CM-2 should still be the primary

	// Write to a node, read from another
	expectedData = "nihao"
	sys.NodeWrite(2, 0, expectedData)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.KillCM(-2) // CM-1 is left, should take over

	// Write to a node, read from another
	expectedData = "dasvidanya"
	sys.NodeWrite(7, 0, expectedData)
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.Exit()
}

func TestMultiFaults(t *testing.T) {
	// Tests scenario of multiple failures and reboots (where there's always at least one primary online)
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()

	// Write to a node, read from another
	expectedData := "WRITE ONE"
	sys.NodeWrite(1, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent := testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.KillCM(-1) // CM-2 is left, should take over during the next write

	// Write to a node, read from another
	expectedData = "WRITE TWO"
	sys.NodeWrite(2, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.RebootCM(-1) // CM-2 should still be the primary
	time.Sleep(1)
	sys.KillCM(-2) // CM-1 is left, should take over during the next write
	expectedData = "WRITE THREE"
	sys.NodeWrite(3, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.RebootCM(-2)
	time.Sleep(1)
	sys.KillCM(-1) // CM-2 is left, should take over during next write
	expectedData = "WRITE FOUR"
	sys.NodeWrite(4, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.RebootCM(-1) // CM-2 should still be the primary
	time.Sleep(1)
	sys.KillCM(-2) // CM-1 is left, should take over during the next write
	expectedData = "WRITE FIVE"
	sys.NodeWrite(5, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}

	sys.RebootCM(-2)
	time.Sleep(1)
	sys.KillCM(-1) // CM-2 is left, should take over during next write
	expectedData = "WRITE SIX"
	sys.NodeWrite(6, 0, expectedData)
	log.Printf("---TESTING IF DATA IS CONSISTENT---")
	data, consistent = testDataConsistent(sys, 0)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", 1, 0, data, expectedData)
	}
	
}
