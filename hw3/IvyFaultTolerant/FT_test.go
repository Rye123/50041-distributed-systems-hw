package main

import (
	"log"
	"testing"
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

func TestSingleFaultAfterElection(t *testing.T) {
	// Tests scenario where primary goes down AFTER it has been initialised as the primary
	tLog := useTempLog(t, true)

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


func TestMultiFaultAndReboot(t *testing.T) {
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
