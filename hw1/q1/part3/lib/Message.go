package lib

type Message struct {
	SrcId     int
	Data      string
	Timestamp ClockVal // Send timestamp of this message.
}

// Returns a total ordering between two messages. A lower SrcId is considered to be "earlier" than a higher SrcId, if the Timestamps are the same.
func MessageLessThan(msg1, msg2 Message) bool {
	if msg1.Timestamp.Compare(msg2.Timestamp) == -1 {
		// if msg1 comes strictly before msg2
		return true
	}
	return msg1.SrcId < msg2.SrcId
}
