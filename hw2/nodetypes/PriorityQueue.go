/**
  A priority queue that uses vector clocks and nodeIds for sorting.

  The element with the LOWEST timestamp will be prioritised.
*/

package nodetypes

import "sync"

type pqueueElem struct {
	nodeId int
	timestamp ClockVal
}

type pqueue struct {
	contents []pqueueElem
	accessLock *sync.Mutex
}

func newPQueue() *pqueue {
	return &pqueue{make([]pqueueElem, 0), &sync.Mutex{}}
}

func (q *pqueue) Length() int {
	return len(q.contents)
}

// Inserts a new element into the queue
func (q *pqueue) Insert(nodeId int, timestamp ClockVal) {
	q.accessLock.Lock(); defer q.accessLock.Unlock()
	newElem := pqueueElem{nodeId, timestamp}
	for i, elem := range q.contents {
		// Iterate until we reach the first element where newElem is < elem
		cmp := compareElems(newElem, elem)
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
func (q *pqueue) Extract() int {
	q.accessLock.Lock(); defer q.accessLock.Unlock()
	if len(q.contents) == 0 {
		panic("Queue is empty!")
	}
	elem := q.contents[0]
	if len(q.contents) == 1 {
		q.contents = make([]pqueueElem, 0)
	} else {
		q.contents = q.contents[1:]
	}
	return elem.nodeId
}

// Peek at the head of a queue, returning the nodeId without popping it
func (q *pqueue) Peek() int {
	if len(q.contents) == 0 {
		panic("Queue is empty!")
	}
	return q.contents[0].nodeId
}

// Returns -1 if el1 < el2, 0 if el1 is not > or < than el2, 1 if el1 > el2
func compareElems(el1, el2 pqueueElem) int {
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
