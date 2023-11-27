package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
)

// Simple struct to contain contents of log
type tempLog struct {
	t *testing.T
	contents []string
	active bool
}

func (tl *tempLog) Write(p []byte) (int, error) {
	tl.contents = append(tl.contents, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (tl *tempLog) Dump() {
	if !tl.active { return }
	for _, line := range tl.contents {
		tl.t.Log(line)
	}
}

// Makes log use a temporary log that is returned -- active is meant as a quick toggle to disable the temp log and just print logs straight to console immediately
func useTempLog(t *testing.T, active bool) *tempLog {
	tempLog := tempLog{t, make([]string, 0), active}
	if !active { return &tempLog }
	log.SetPrefix("")
	log.SetFlags(log.Ltime)
	log.SetOutput(&tempLog)

	return &tempLog
}

// Returns (data, true) if the data in the entire system is consistent, otherwise returns ("", false)
// Note that this is NOT threadsafe -- node reads are done sequentially.
func testDataConsistent(s *System, pageId PageId) (string, bool) {
	if len(s.nodes) == 0 {
		panic("No nodes!")
	}
	overall_data := ""
	is_data_set := false
	
	for node_id := range s.nodes {
		data := s.NodeRead(node_id, pageId)
		if !is_data_set {
			overall_data = data
			is_data_set = true
			continue
		}

		if data != overall_data {
			return "", false
		}
	}
	return overall_data, true
}

func TestStandardUsage(t *testing.T) {
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

func TestMultiWrite(t *testing.T) {
	tLog := useTempLog(t, false)
	
	sys := NewSystem(20)
	sys.Init()

	// Write to multiple nodes at once -- since we don't enforce any timestamps, we at the least expect a CONSISTENT result
	counter := 0
	var writeWg sync.WaitGroup
	pageId := PageId(69)
	for nodeId := range sys.nodes {
		writeWg.Add(1)
		nodeId := nodeId
		go func(c int) {
			defer writeWg.Done()
			sys.NodeWrite(nodeId, pageId, fmt.Sprintf("data%d", c))
		}(counter)
		counter++
	}

	writeWg.Wait()

	// Expect a consistent result
	data, consistent := testDataConsistent(sys, pageId)
	if !consistent {
		tLog.Dump()
		t.Fatalf("Data not consistent.")
	}

	// Expect data to at the very least contain "data"
	if !strings.Contains(data, "data") {
		tLog.Dump()
		t.Fatalf("Final data is %v, expected it to contain the word \"data\".", data)
	}
}
