package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestFaultless_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()

	pageId := PageId(69)

	for count := 1; count <= 3; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}

	for count := 4; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}
	
}


func TestSingleFault_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()
	
	pageId := PageId(69)

	for count := 1; count <= 3; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}

	sys.KillCM(NodeId(-1))

	for count := 4; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}
	
}


func TestSingleFaultAndReboot_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()
	
	pageId := PageId(69)

	for count := 1; count <= 3; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}

	sys.KillCM(NodeId(-1))

	for count := 4; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}

		if count == 5 {
			sys.RebootCM(NodeId(-1))
		}
	}
	
}

func TestSingleFaultRandom_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()
	
	pageId := PageId(69)
	randMilli := rand.Intn(1100 - 900) + 900
	timer := time.NewTimer(time.Millisecond * time.Duration(randMilli))
	go func() {
		<-timer.C
		sys.KillCM(NodeId(-1))
	}()

	for count := 1; count <= 3; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}

	for count := 4; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
	}
	
}



func TestMultiFaultsOfPrimary_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()
	
	pageId := PageId(69)
	prevPri := InvalidNodeId
	currentPri := NodeId(-1)

	// Initial run
	expectedData := fmt.Sprintf("asdf%d", 1)
	nodeId := NodeId(1)
	sys.NodeWrite(nodeId, pageId, expectedData)
	data, consistent := testDataConsistent(sys, pageId)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}
	if data != expectedData {
		tLog.Dump()
		t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
	}
	sys.KillCM(currentPri)
	currentPri = NodeId(-2)

	for count := 2; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}
		// "Switch" CMs
		prevPri = currentPri
		if currentPri == NodeId(-1) {
			currentPri = NodeId(-2)
		} else {
			currentPri = NodeId(-1)
		}
		sys.RebootCM(currentPri)
		time.Sleep(10 * time.Millisecond)
		sys.KillCM(prevPri)
	}
	
}


func TestMultiFaultsOfPrimaryAndBackup_PERF(t *testing.T) {
	tLog := useTempLog(t, false)

	sys := NewSystem(10)
	sys.Init()
	
	pageId := PageId(69)
	killedNodeId := InvalidNodeId

	for count := 2; count <= 6; count++ {
		expectedData := fmt.Sprintf("asdf%d", count)
		nodeId := NodeId(count)
		sys.NodeWrite(nodeId, pageId, expectedData)
		data, consistent := testDataConsistent(sys, pageId)
		if !consistent {
			tLog.Dump()
			t.Fatalf("Data not consistent.")
		}
		if data != expectedData {
			tLog.Dump()
			t.Fatalf("Read from N%d for P%d returned %v, expected %v", nodeId, pageId, data, expectedData)
		}

		// Reboot the killed node
		if killedNodeId != InvalidNodeId {
			sys.RebootCM(killedNodeId)
			time.Sleep(10 * time.Millisecond)
		}

		// Randomly kill a node
		choice := rand.Float32()
		if choice < 0.5 {
			killedNodeId = NodeId(-1)
		} else {
			killedNodeId = NodeId(-2)
		}
		sys.KillCM(killedNodeId)
	}
	
}
