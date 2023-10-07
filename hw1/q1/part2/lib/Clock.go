package lib

import "math"

type ClockVal int

// Returns the greater of two ClockVals
func MaxClockValue(c1, c2 ClockVal) ClockVal {
	return ClockVal(math.Max(float64(c1), float64(c2)))
}
