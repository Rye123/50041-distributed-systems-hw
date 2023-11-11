package clock

import (
	"math"
)

type ClockVal struct {
	values map[int]int // Maps a node ID to its clock value.
}

func NewClockVal(nodeIds []int) ClockVal {
	values := make(map[int]int, len(nodeIds))

	for _, nodeId := range nodeIds {
		values[nodeId] = 0
	}
	return ClockVal{values}
}

func ManualClock(values map[int]int) ClockVal {
	return ClockVal{values}
}

// Returns:
// - 0  if both clock values are CONCURRENT (i.e. equal or neither gt/lt)
// - 1  if c1 is STRICTLY > c2
// - -1 if c2 is STRICTLY < c2
func (c1 ClockVal) Compare(c2 ClockVal) int {
	retVal := 0
	for nodeId := range c1.values {
		if _, exists := c2.values[nodeId]; !exists {
			panic("Compare error: c1 not of same structure as c2")
		}

		if c1.values[nodeId] > c2.values[nodeId] {
			if retVal == -1 {
				// previously, c1[i] < c2[i]. Hence they must be concurrent
				return 0
			}
			retVal = 1
		} else if c1.values[nodeId] < c2.values[nodeId] {
			if retVal == 1 {
				// previously, c1[i] > c2[i]. Hence they must be concurrent
				return 0
			}
			retVal = -1
		}
	}

	// If retVal == -1 / 1, then there exists at least 1 value
	// that is less than/equal than the corresponding value.
	// If retVal == 0, then every value was equal.
	return retVal
}

// Returns a copy of the clock value.
func (c ClockVal) Clone() ClockVal {
	values := make(map[int]int, len(c.values))
	for nodeId := range c.values {
		values[nodeId] = c.values[nodeId]
	}
	return ClockVal{values}
}

// Returns a new clock value, with the relevant node incremented.
func (c ClockVal) Increment(nodeId int, increment int) ClockVal {
	clkVal := c.Clone()
	clkVal.values[nodeId] += increment
	return clkVal
}

// Returns the elementwise max of two clock values
func MaxClockValue(c1, c2 ClockVal) ClockVal {
	clkVal := c1.Clone()
	for nodeId := range c1.values {
		v1 := c1.values[nodeId]
		v2, exists := c2.values[nodeId]
		if !exists {
			panic("MaxClockValue error: c1 not of same structure as c2.")
		}
		clkVal.values[nodeId] = int(math.Max(float64(v1), float64(v2)))
	}
	return clkVal
}
