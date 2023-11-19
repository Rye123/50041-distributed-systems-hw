/**
  A priority queue that uses vector clocks and nodeIds for sorting.
  - This is specific for the Voting Protocol, since we need to store the election ID too.

  The element with the LOWEST timestamp will be prioritised.
*/

package nodetypes

import "sync"

type pqueueVPElem struct {
	nodeId int
	electionId string
	timestamp ClockVal
}

type pqueueVP struct {
	contents []pqueueVPElem
	accessLock *sync.Mutex
}

func newPQueueVP() *pqueueVP {
	return &pqueueVP{make([]pqueueVPElem, 0), &sync.Mutex{}}
}

func (q *pqueueVP) Length() int {
	return len(q.contents)
}

// Inserts a new element into the queue
func (q *pqueueVP) Insert(nodeId int, electionId string, timestamp ClockVal) {
	q.accessLock.Lock(); defer q.accessLock.Unlock()
	newElem := pqueueVPElem{nodeId, electionId, timestamp}
	for i, elem := range q.contents {
		// Iterate until we reach the first element where newElem is < elem
		cmp := compareElemsVP(newElem, elem)
		if cmp == -1 {
			q.contents = append(q.contents[:i+1], q.contents[i:]...)
			q.contents[i] = newElem
			return
		} else if cmp == 0 {
			panic("Concurrent timestamps with the SAME machine ID. Will cause deadlock.")
		}
	}
	q.contents = append(q.contents, newElem)
}

// Pops the head of the queue, returning the nodeId
func (q *pqueueVP) Extract() int {
	q.accessLock.Lock(); defer q.accessLock.Unlock()
	if len(q.contents) == 0 {
		panic("Queue is empty!")
	}
	elem := q.contents[0]
	if len(q.contents) == 1 {
		q.contents = make([]pqueueVPElem, 0)
	} else {
		q.contents = q.contents[1:]
	}
	return elem.nodeId
}

func (q *pqueueVP) ExtractElem() pqueueVPElem {
	q.accessLock.Lock(); defer q.accessLock.Unlock()
	if len(q.contents) == 0 {
		panic("Queue is empty!")
	}
	elem := q.contents[0]
	if len(q.contents) == 1 {
		q.contents = make([]pqueueVPElem, 0)
	} else {
		q.contents = q.contents[1:]
	}
	return elem
}

// Peek at the head of a queue, returning the nodeId without popping it
func (q *pqueueVP) Peek() int {
	if len(q.contents) == 0 {
		panic("Queue is empty!")
	}
	return q.contents[0].nodeId
}

// Returns -1 if el1 < el2, 0 if el1 is not > or < than el2, 1 if el1 > el2
func compareElemsVP(el1, el2 pqueueVPElem) int {
	if el1.timestamp < el2.timestamp {
		return -1
	} else if el1.timestamp > el2.timestamp {
		return 1
	}

	// Otherwise, fall back to machine ID -- lower ID should have a lower "timestamp"
	if el1.nodeId < el2.nodeId {
		return -1
	} else if el1.nodeId > el2.nodeId {
		return 1
	}
	return 0
}
