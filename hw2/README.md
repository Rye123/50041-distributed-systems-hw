# HW2

We follow the following scenario: We have a **shared** memory space in the form of a struct `nodetypes.SharedMemory`, which can be modified by an interface `Node`.
- This struct provides the functions `EnterCS` and `ExitCS`, which modify the shared memory space.
- The methods provided `panic()` when a node enters the "critical section" while a previous node has yet to call `ExitCS()`.
- This forces the implementer of `Node` to provide their own mutual exclusion algorithms, allowing us a common interface to test implementations of Lamport's Shared Priority Queue, Ricart and Agrawala's Optimisation, and the Voting Protocol.
- A simple implementation can be seen in `nodetypes.NaiveNode`, which simply acquires and releases the lock ignoring everyone else. This would breach the safety condition, resulting in a `panic()`.

The `Orchestrator` allows us a convenient interface to simulate various test cases. By using it along with the `testing` package, we can create a readable and non-verbose series of tests.
- An example is in `Orchestrator_test.go`, which tests the orchestrator using the above naive implementation and ensures that it successfully detects the resulting breach of the safety condition.

## Usage

### Testing (Grading)
The tester code for grading is in `DEMO_test.go`.
- For these tests, the default node count is 10 and the default time each node spents in the critical section is 100 milliseconds.
  - This can be changed with `DEMO_NODE_COUNT` and `DEMO_CS_DELAY` in `DEMO_test.go`.
- Note that the output is in **physical clock order** based on the time `log.Printf` was called -- **this may not match the action order of events**. The true order is in logical clock order.

#### Lamport's Shared Priority Queue
To run this, do:
```bash
go test --run TestLamport_DEMO
```

This should have output like:
```
TestLamport_DEMO initialised with 10 nodes and CS delay 100ms.
2023/11/20 15:32:32 N1: 47: Lock acquired. Entering CS. Queue: [{1 1} {6 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N1: 49: Exiting CS. Lock Released. Queue: [{2 1} {6 1} {7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N2: 60: Lock acquired. Entering CS. Queue: [{2 1} {6 1} {7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N2: 60: Exiting CS. Lock Released. Queue: [{6 1} {7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N6: 62: Lock acquired. Entering CS. Queue: [{6 1} {7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N6: 62: Exiting CS. Lock Released. Queue: [{7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N7: 64: Lock acquired. Entering CS. Queue: [{7 1} {8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N7: 64: Exiting CS. Lock Released. Queue: [{8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N8: 66: Lock acquired. Entering CS. Queue: [{8 1} {9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N8: 66: Exiting CS. Lock Released. Queue: [{9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N9: 68: Lock acquired. Entering CS. Queue: [{9 1} {4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N9: 68: Exiting CS. Lock Released. Queue: [{4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:32 N4: 70: Lock acquired. Entering CS. Queue: [{4 4} {5 4} {3 5} {0 7}]
2023/11/20 15:32:33 N4: 70: Exiting CS. Lock Released. Queue: [{5 4} {3 5} {0 7}]
2023/11/20 15:32:33 N5: 72: Lock acquired. Entering CS. Queue: [{5 4} {3 5} {0 7}]
2023/11/20 15:32:33 N5: 72: Exiting CS. Lock Released. Queue: [{3 5} {0 7}]
2023/11/20 15:32:33 N3: 74: Lock acquired. Entering CS. Queue: [{3 5} {0 7}]
2023/11/20 15:32:33 N3: 74: Exiting CS. Lock Released. Queue: [{0 7}]
2023/11/20 15:32:33 N0: 76: Lock acquired. Entering CS. Queue: [{0 7}]
2023/11/20 15:32:33 N0: 76: Exiting CS. Lock Released. Queue: []
2023/11/20 15:32:33 Orchestrator: Shutdown
PASS
ok      1005129_RYAN_TOH/hw2    1.377s
```

To interpret the above output:
- Each line is arranged this way: `[Physical Date Time] [Node]: [Logical Clock]: [Action]. Queue: [Node's local Queue]`.
  - To interpret the queue: `[{node request_timestamp}]`
  - This queue will be ordered with the smallest timestamp. In the case of concurrent timestamps, the lower ID will have the 'smaller' timestamp.

#### Ricart and Agrawala's Optimisation
To run this, do:
```bash
go test --run TestRicart_DEMO
```

This should have output like:
```
TestRicart_DEMO initialised with 10 nodes and CS delay 100ms.
2023/11/20 15:33:47 N0: 30: Lock acquired. Entering CS. Queue: [{0 1} {2 1} {3 1} {4 1} {5 1} {6 1} {7 1} {8 1} {9 1}]
2023/11/20 15:33:48 N0: 31: Exiting CS. Lock Released. Queue: [{2 1} {3 1} {4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N2: 33: Lock acquired. Entering CS. Queue: [{2 1} {3 1} {4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N2: 33: Exiting CS. Lock Released. Queue: [{3 1} {4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N3: 37: Lock acquired. Entering CS. Queue: [{3 1} {4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N3: 37: Exiting CS. Lock Released. Queue: [{4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N4: 39: Lock acquired. Entering CS. Queue: [{4 1} {5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N4: 39: Exiting CS. Lock Released. Queue: [{5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N5: 41: Lock acquired. Entering CS. Queue: [{5 1} {6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N5: 41: Exiting CS. Lock Released. Queue: [{6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N6: 43: Lock acquired. Entering CS. Queue: [{6 1} {7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N6: 43: Exiting CS. Lock Released. Queue: [{7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N7: 45: Lock acquired. Entering CS. Queue: [{7 1} {8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N7: 45: Exiting CS. Lock Released. Queue: [{8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N8: 47: Lock acquired. Entering CS. Queue: [{8 1} {9 1} {1 6}]
2023/11/20 15:33:48 N8: 47: Exiting CS. Lock Released. Queue: [{9 1} {1 6}]
2023/11/20 15:33:48 N9: 49: Lock acquired. Entering CS. Queue: [{9 1} {1 6}]
2023/11/20 15:33:48 N9: 49: Exiting CS. Lock Released. Queue: [{1 6}]
2023/11/20 15:33:48 N1: 51: Lock acquired. Entering CS. Queue: [{1 6}]
2023/11/20 15:33:49 N1: 51: Exiting CS. Lock Released. Queue: []
2023/11/20 15:33:49 Orchestrator: Shutdown
PASS
ok      1005129_RYAN_TOH/hw2    1.330s
```

To interpret the above output:
- Each line is arranged this way: `[Physical Date Time] [Node]: [Logical Clock]: [Action]. Queue: [Node's local Queue]`.
  - To interpret the queue: `[{node request_timestamp}]`
  - This queue will be ordered with the smallest timestamp. In the case of concurrent timestamps, the lower ID will have the 'smaller' timestamp.
  
#### Voting Protocol
To run this, do:
```bash
go test --run TestVoting_DEMO
```

Due to the difference in protocol there's far more messages in the output, hence it has been omitted from this `README`.
- In general, each line has a **different** format from above.
- The format is: `[Physical Date Time] [[Logical Clock]] - [Node]: [Action]`.
- For a single node, the rough sequence of events (rearragned based on logical clock) will be:
  ```
  2023/11/20 15:39:23 [13] - N4: Start request
  2023/11/20 15:39:23 [4] - N8: VOTED for N4.
  2023/11/20 15:39:23 [13] - N4: Received VOTE from N8. Voters: [8]
  ...
  (Sequence for a rescind)
  2023/11/20 15:39:23 [14] - N8: ATTEMPT RESCIND from N4.
  2023/11/20 15:39:23 [16] - N4: Received RESCIND from N8. Released N8's vote.
  2023/11/20 15:39:23 [19] - N8: Received RELEASE from N4. VOTED for N0.
  ...
  (Further votes)
  2023/11/20 15:39:24 [77] - N4: Received VOTE from N0. Voters: [0]
  2023/11/20 15:39:24 [79] - N4: Received VOTE from N1. Voters: [1 0]
  2023/11/20 15:39:24 [84] - N4: Received VOTE from N3. Voters: [1 3 5 0 7 8]
  2023/11/20 15:39:24 [84] - N4: Lock acquired. Entering CS. Voters: [0 7 8 5 1 3]
  ...
  (Receiving votes AFTER entering CS -- here we simply release those votes)
  2023/11/20 15:39:24 [86] - N4: Received RELEASE from N2. VOTED for N4.
  2023/11/20 15:39:24 [87] - N4: Received late VOTE from N4.
  2023/11/20 15:39:24 [93] - N4: Received RELEASE from N4. VOTED for N5.
  ...
  (Completed CS)
  2023/11/20 15:39:24 [97] - N4: Lock released
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N1.
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N3.
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N7.
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N8.
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N5.
  2023/11/20 15:39:24 [97] - N4: Sent RELEASE to N0.
  ```


### Printing the Table
Running `main.go` prints the timing calculations.
```bash
go run .
```
- This calculates the time taken for x nodes to enter the CS, out of N nodes.
- N is 10, and each node spends 100 milliseconds in the CS.

```
---LAMPORT'S SHARED PRIORITY QUEUE---
Progress: ||||||||||
NODES ENTERING | TIME TAKEN
             1 | 146ms
             2 | 218ms
             3 | 333ms
             4 | 428ms
             5 | 573ms
             6 | 696ms
             7 | 778ms
             8 | 920ms
             9 | 1.017s
            10 | 1.112s

---RICART AND AGRAWALA'S OPTIMISATION---
Progress: ||||||||||
NODES ENTERING | TIME TAKEN
             1 | 110ms
             2 | 222ms
             3 | 348ms
             4 | 445ms
             5 | 553ms
             6 | 683ms
             7 | 772ms
             8 | 920ms
             9 | 1.033s
            10 | 1.205s

---VOTING PROTOCOL---
Progress: ||||||||||
NODES ENTERING | TIME TAKEN
             1 | 109ms
             2 | 220ms
             3 | 331ms
             4 | 441ms
             5 | 554ms
             6 | 667ms
             7 | 791ms
             8 | 885ms
             9 | 996ms
            10 | 1.106s
```

We have the following table:


| Nodes Entering                                      | 1   | 2   | 3   | 4   | 5   | 6   | 7   | 8   | 9    | 10   |
| --------------------------------------------------- | --- | --- | --- | --- | --- | --- | --- | --- | ---- | ---- |
| Lamport Shared Priority Queue: Time Taken (ms)      | 146 | 218 | 333 | 428 | 573 | 696 | 778 | 920 | 1017 | 1112 |
| Ricart and Agrawala's Optimisation: Time Taken (ms) | 110 | 222 | 348 | 445 | 553 | 683 | 772 | 920 | 1033 | 1205 |
| Voting Protocol: Time Taken (ms)                    | 109 | 220 | 331 | 441 | 554 | 667 | 791 | 885 | 996  | 1106 | 


### Testing (Development)
Our main tester code is in `Implementations_test.go`. This runs the tests without any logging of node output, which is less useful for marking.
- The default settings is to run 100 nodes, and for each node to spend 100 milliseconds in the critical section.
  - This can be **changed** with `TEST_NODE_COUNT` and `TEST_CS_DELAY` in `Implementations_test.go`.
- Two tests are being run:
  - `TestStandard_*` spawns N goroutines that make all N nodes concurrently attempt to enter the CS.
  - `TestHalfConcurrent_*` spawns N/2 goroutines that makes N/2 nodes concurrent attempt to enter the CS.
- Orchestrator tests aren't directly related to the assignment, they simply run the Orchestrator with `NaiveNode`s to ensure that the Orchestrator reports any safety violations.

To run these, do:
```bash 
go test . -v --short
```

This should have the following output:
```
=== RUN   TestLamport_DEMO
    DEMO_test.go:28: TestLamport_DEMO skipped (Short Mode).
--- SKIP: TestLamport_DEMO (0.00s)
=== RUN   TestRicart_DEMO
    DEMO_test.go:63: TestRicart_DEMO skipped (Short Mode).
--- SKIP: TestRicart_DEMO (0.00s)
=== RUN   TestVoting_DEMO
    DEMO_test.go:98: TestVoting_DEMO skipped (Short Mode).
--- SKIP: TestVoting_DEMO (0.00s)
=== RUN   TestStandard_Lamport
--- PASS: TestStandard_Lamport (23.56s)
=== RUN   TestHalfConcurrent_Lamport
--- PASS: TestHalfConcurrent_Lamport (10.57s)
=== RUN   TestStandard_Ricart
--- PASS: TestStandard_Ricart (20.32s)
=== RUN   TestHalfConcurrent_Ricart
--- PASS: TestHalfConcurrent_Ricart (8.38s)
=== RUN   TestStandard_Voting
--- PASS: TestStandard_Voting (11.07s)
=== RUN   TestHalfConcurrent_Voting
--- PASS: TestHalfConcurrent_Voting (5.57s)
=== RUN   TestOrchestratorNoError
--- PASS: TestOrchestratorNoError (0.00s)
=== RUN   TestOrchestratorSameNodeEnterTwice
--- PASS: TestOrchestratorSameNodeEnterTwice (0.00s)
=== RUN   TestOrchestratorTwoNodesEnterCS
--- PASS: TestOrchestratorTwoNodesEnterCS (0.00s)
=== RUN   TestOrchestratorRandomNodeExit
--- PASS: TestOrchestratorRandomNodeExit (0.00s)
=== RUN   TestOrchestratorNodeEnterAnotherExit
--- PASS: TestOrchestratorNodeEnterAnotherExit (0.00s)
PASS
ok      1005129_RYAN_TOH/hw2    79.783s
```







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

### Voting Protocol
The prior implementations heavily relied on the fact that "votes" (i.e. request acknowledgements) could not be rescinded. Hence, it was safe to handle each incoming message concurrently regardless of the message type, relying on mutex locks to ensure safety.

In the Voting Protocol (with deadlock prevention), we needed to be able to rescind a vote, which meant that if the above implementation of concurrent message handling were to be naively used, there would be separate goroutines handling incoming requests and releases. This made it far more difficult as any mutex lock over the node's vote would need to be *shared* among goroutines, destroying the purpose of the mutex lock in the first place.

Hence, I had a separate goroutine that handled requests and releases sequentially (with a channel). A modified version of the local priority queue was used to keep track of the requests that had yet to be voted for. 

To account for late messages (e.g. late `VOTE`s and late `RESCIND`s), each 'election' (where a node requested to enter the CS) had a unique election ID. 
- Further, since a node could receive a `RESCIND` before the actual `VOTE`, a `voteSession` manager was used to keep track of the actual vote status of nodes.
