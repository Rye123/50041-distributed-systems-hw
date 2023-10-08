package lib

import (
	"testing"
)

// Returns a ClockVal with the given values, rather than using IDs.
// This is for testing purposes -- the real NewClockVal function takes in a list
// of node Ids, instead of a list of values.
func newClockVal(values []int) ClockVal {
	nodeIds := make([]int, len(values))
	for key := range values {
		nodeIds[key] = key
	}
	clkVal := NewClockVal(nodeIds)
	for nodeId, v := range values {
		clkVal.values[nodeId] = v
	}
	return clkVal
}

func TestCompare(t *testing.T) {
	// Test equality concurrency
	c1, c2 := newClockVal([]int{0, 0, 0}), newClockVal([]int{0, 0, 0})
	if c1.Compare(c2) != 0 {
		t.Fatalf("{0,0,0} not conc with {0,0,0}")
	}

	// Test greater than
	c1, c2 = newClockVal([]int{1, 0, 0}), newClockVal([]int{0, 0, 0})
	if c1.Compare(c2) != 1 {
		t.Fatalf("{1,0,0} not > {0,0,0}")
	}

	// Test less than
	c1, c2 = newClockVal([]int{1, 0, 0}), newClockVal([]int{2, 0, 0})
	if c1.Compare(c2) != -1 {
		t.Fatalf("{1,0,0} not < {2,0,0}")
	}

	// Test mixed concurrency
	c1, c2 = newClockVal([]int{1, 2, 3}), newClockVal([]int{3, 2, 1})
	if c1.Compare(c2) != 0 {
		t.Fatalf("{1,2,3} not conc with {3,2,1}")
	}

	// Test greater than
	c1, c2 = newClockVal([]int{1, 1, 1}), newClockVal([]int{0, 0, 0})
	if c1.Compare(c2) != 1 {
		t.Fatalf("{1,1,1} not > {0,0,0}")
	}

	// Test greater than
	c1, c2 = newClockVal([]int{9, 8, 7, 6}), newClockVal([]int{1, 2, 3, 4})
	if c1.Compare(c2) != 1 {
		t.Fatalf("{9,8,7,6} not > {1,2,3,4}")
	}

	// Test greater than (with others equal)
	c1, c2 = newClockVal([]int{9, 10, 8}), newClockVal([]int{9, 9, 8})
	if c1.Compare(c2) != 1 {
		t.Fatalf("{9,10,8} not > {9,9,8}")
	}
}
