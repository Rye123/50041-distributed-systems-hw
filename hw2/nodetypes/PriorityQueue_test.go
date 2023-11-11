package nodetypes

import (
	"1005129_RYAN_TOH/hw2/clock"
	"testing"
)

func manualClockVal(vals []int) clock.ClockVal {
	values := make(map[int]int)
	for i, val := range vals {
		values[i] = val
	}
	return clock.ManualClock(values)
}

func initPriorityQueue(elems []pqueueElem) pqueue {
	q := newPQueue()
	for _, elem := range elems {
		q.Insert(elem.nodeId, elem.timestamp)
	}
	return *q
}

func dumpPriorityQueue(q pqueue) []int {
	ret := make([]int, 0)
	for q.Length() != 0 {
		ret = append(ret, q.Extract())
	}
	return ret
}

func TestPriorityQueueSimple(t *testing.T) {
	elems := []pqueueElem{
		pqueueElem{0, manualClockVal([]int{1})},
		pqueueElem{0, manualClockVal([]int{3})},
		pqueueElem{0, manualClockVal([]int{2})},
	}
	expectedOrder := []int{0, 0, 0}
	queue := initPriorityQueue(elems)
	resultOrder := dumpPriorityQueue(queue)

	if len(expectedOrder) != len(resultOrder) {
		t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
	}

	for i := range expectedOrder {
		if expectedOrder[i] != resultOrder[i] {
			t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
		}
	}
}

func TestPriorityQueueSameElemInVecClock(t *testing.T) {
	elems := []pqueueElem{
		pqueueElem{0, manualClockVal([]int{0, 0, 0})},
		pqueueElem{1, manualClockVal([]int{2, 0, 0})},
		pqueueElem{2, manualClockVal([]int{1, 0, 0})},
	}
	expectedOrder := []int{0, 2, 1}
	queue := initPriorityQueue(elems)
	resultOrder := dumpPriorityQueue(queue)

	if len(expectedOrder) != len(resultOrder) {
		t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
	}

	for i := range expectedOrder {
		if expectedOrder[i] != resultOrder[i] {
			t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
		}
	}
}


func TestPriorityQueueDiffElemInVecClock(t *testing.T) {
	elems := []pqueueElem{
		pqueueElem{0, manualClockVal([]int{1, 0, 0})},
		pqueueElem{1, manualClockVal([]int{1, 5, 3})},
		pqueueElem{2, manualClockVal([]int{1, 0, 3})},
	}
	expectedOrder := []int{0, 2, 1}
	queue := initPriorityQueue(elems)
	resultOrder := dumpPriorityQueue(queue)

	if len(expectedOrder) != len(resultOrder) {
		t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
	}

	for i := range expectedOrder {
		if expectedOrder[i] != resultOrder[i] {
			t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
		}
	}
}

func TestPriorityQueueResolveWithMachineID(t *testing.T) {
	elems := []pqueueElem{
		pqueueElem{0, manualClockVal([]int{1, 0, 0})},
		pqueueElem{1, manualClockVal([]int{0, 1, 0})},
		pqueueElem{2, manualClockVal([]int{0, 0, 1})},
	}
	expectedOrder := []int{0, 1, 2}
	queue := initPriorityQueue(elems)
	resultOrder := dumpPriorityQueue(queue)

	if len(expectedOrder) != len(resultOrder) {
		t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
	}

	for i := range expectedOrder {
		if expectedOrder[i] != resultOrder[i] {
			t.Fatalf("Expected: %v, Got: %v", expectedOrder, resultOrder)
		}
	}
}
