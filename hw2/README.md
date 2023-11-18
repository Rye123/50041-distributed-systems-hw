# HW2

We follow the following scenario: We have a **shared** memory space in the form of a struct `nodetypes.SharedMemory`, which can be modified by an interface `Node`.
- This struct provides the functions `EnterCS` and `ExitCS`, which modify the shared memory space.
- The methods provided `panic()` when a node enters the "critical section" while a previous node has yet to call `ExitCS()`.
- This forces the implementer of `Node` to provide their own mutual exclusion algorithms, allowing us a common interface to test implementations of Lamport's Shared Priority Queue, Ricart and Agrawala's Optimisation, and the Voting Protocol.
- A simple implementation can be seen in `nodetypes.NaiveNode`, which simply acquires and releases the lock ignoring everyone else. This would breach the safety condition, resulting in a `panic()`.

The `Orchestrator` allows us a convenient interface to simulate various test cases. By using it along with the `testing` package, we can create a readable and non-verbose series of tests.
- An example is in `Orchestrator_test.go`, which tests the orchestrator using the above naive implementation and ensures that it successfully detects the resulting breach of the safety condition.

## Implementation Notes
### Lamport's Shared Priority Queue
For this, I utilised an internal priority queue for each node -- which to avoid race conditions had its own internal mutex lock.

The node itself had a single goroutine that listened for messages from other nodes. Instructions for the node to acquire the lock were given as separate goroutines.
- When a node received a message, it handles the message in yet another goroutine. This created the possibility of race conditions when modifying internal variables.
- Hence, I used more mutex locks to prevent race conditions.
  - Each node had its own internal counter for the number of responses received for the most recent request. Since messages were handled concurrently, there could be multiple handler goroutines for the numerous `REQ_ACK`s received -- hence a mutex lock was used to prevent a race condition here.
  - Each node, when considering a request from itself, differentiated between an ONGOING request and a PENDING request.
	- An ONGOING request was a request from the node itself that was still in its queue.
	- A PENDING request was a request that had yet to receive all responses for it.
   - To ensure no race conditions, locks were used for both types of requests.

### Ricart and Agrawala's Optimisation
Again, I used the above implemented internal priority queue, and each node had a single goroutine that listened for messages from other nodes.
- Here, since an ONGOING request was also a PENDING request (using the terminology from above), we could reduce the tracking of this to a single mutex lock.
- However, for modification of the response count, a mutex lock was still needed.

## Testing
	Our main test involves simultaneously creating 100 nodes. Then, we initialise 100 goroutines for each node to sequentially enter and exit the critical section.
