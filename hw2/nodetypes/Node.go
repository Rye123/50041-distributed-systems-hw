package nodetypes

type ClockVal int
func MaxClockVal(c1, c2 ClockVal) ClockVal {
	if c1 > c2 {
		return c1
	}
	return c2
}

// Interface for a node that uses SharedMemory (see SharedMemory.go)
type Node interface {
	Init() error
	AcquireLock()
	ReleaseLock()
	Shutdown() error
}
