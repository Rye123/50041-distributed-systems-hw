# PSET1 Question 2

The bulk of the code is in the `lib` package, which implements the Bully algorithm among $N$ nodes that synchronise a string value using Go channels.
- Each node runs concurrently, using goroutines.
- Synchronisation is done using channels.
  - Each node has a data channel and control message channel that it constantly reads from.
  - Any incoming message from another node is sent to these channels.
- Each node starts an election upon initialisation. Each node can have only one of its own elections going on at a time (hence in the worst case, we have $N$ elections occurring, with $N$ nodes).
- A "fault" is simulated by a node simply not communicating anymore. The node stops sending messages and stops responding to messages, while messages received through its channels are simply dropped.
  - This is detected on other nodes' ends by a timeout.
  - This is because if we were to kill a node by closing its channels, those channels cannot be reopened by nature of the Go language.
  
## Usage

### Grading

The `Demo_test.go` file contains most of the proofs necessary to show that my code addresses the necessary considerations mentioned in the question. Skip to **Considerations** to see how to run each test.

These tests use 10 nodes by default, however this can be changed in `DEMO_NODECOUNT` (at the cost of much more output messages).

In general, the convention is:
```bash
go test ./lib -v -run Test_[Test Name]_DEMO
```

Refer to **Considerations** for how to run each specific test.

### Manual Testing

For manual testing, please use `main.go`. Currently, it runs the system with 10 nodes, and randomly does one of the following actions every few seconds:
- Updates the coordinator with a new value.
- Randomly selects a node -- if it is alive, kill it; otherwise, revive it.
- Kills the coordinator.

```bash
go run main.go
```

Alternatively, for general testing (without the log output printed in `Demo_test.go`), run:
```bash
go test ./lib -v -short
```

This would skip the tests in the `Demo_test.go` file, which are for demonstrations.
- It isn't recommended to remove the `short` flag, as the `Demo_test.go` tests print output regardless of whether or not the test is successful, making the output more lengthy.

Running all the tests could take up to 5 minutes to complete due to the number of system tests being used. Hence, I recommend manually running the tests in the `Demo_test.go` file, each of which are specified under the Considerations section. 

## Considerations
The various considerations given in the question are resolved using Go tests. Here, I list out most of the tests, along with the respective consideration resolved.

For reference, here are the considerations:
1. Implement the protocol of joint synchronization and election via Go.
2. Simulate the worst-case and best-case situation.
3. Consider the case where a Goroutine fails during the election, yet the routine was alive when the election started.
   a. The newly-elected coordinator fails when announcing it has won the election to all nodes. Thus, some nodes received the information about the new coordinator, but some others do not.
   b. The failed node is not the newly elected coordinator.
4. Multiple Goroutines start the election process simulataneously.
5. An arbitrary node silently leaves the network (departed node can be the coordinator or a non-coordinator).

I use an `Orchestrator` class in `lib`, which automatically creates the necessary nodes and provides helper functions to manipulate the system.

### Basic Initialisation Tests
1. Start up $N$ nodes, wait for election to complete.
2. Ensure coordinator node is the node with the highest ID, $N-1$.

This satisfies **consideration 4**. Our implementation has *every* new node start its own election by default. Hence, simply by running these tests we have multiple goroutines starting the election process simultaneously, in this test we show that this results in the highest ID becoming the coordinator regardless of the number of nodes starting an election.

This also partially satisfies **consideration 1**, by handling a basic election. The detection of a failed coordinator is later shown in the Basic Crash and Reboot Tests.

Finally, this also satisfies the *worst case* of the Bully Algorithm (hence satisfying part of **consideration 2**). In the worst case, *every node* detects that the coordinator has failed and begins their own election. In our implementation, every node starts an election upon joining, which simulates this worst case.
- For the textbook worst case and best case, refer to the **Miscellaneous Tests** below.

To directly test this:
```bash
go test ./lib -v -run Test_BasicInit_DEMO
```

All nodes should begin elections upon initialisation, before the node with the highest ID wins.

### Complex Initialisation Test
Here, we manually modify multiple nodes to consider themselves coordinators. Ideally, the system should fix itself once these coordinators start broadcasting -- higher ID nodes should attempt to elect themselves upon receiving a data broadcast from lower ID nodes. 

### Basic Synchronisation Tests
1. Start up $N$ nodes, wait for election to complete.
2. Update coordinator node $N-1$ with a new value, and wait for value to be propagated (i.e. RTT + default timeout)
3. Ensure that ALL nodes have that modified value.

We also have another test which replaces multiple nodes' values (simulating data corruption), and ensuring that the coordinator's value is the one that replaces all other values. This test is done in `Test_BasicSync_25Nodes_Corrupted`.

This satisfies the other part of **consideration 1**, where we have joint synchronisation of values across the system. The additional test that 'corrupts' values further validates this, as the coordinator's value is taken as the sole source of truth and is propagated through the system.

To directly test this:
```bash
go test ./lib -v -run Test_BasicSync_DEMO
```

There will first be an election. Then, the system will update the value of the coordinator, and after this update, the system will report the values of each node (showing that the value is propagated as expected).

### Basic Crash and Reboot Tests
1. Start up $N$ nodes, wait for election to complete.
2. Shut down coordinator node, wait for re-election to complete.
3. Ensure that the node with the next highest ID is selected (i.e. $N-2$).
4. Reboot original coordinator, wait for re-election to complete.
5. Ensure that node with the highest ID is selected (i.e. $N-1$)

In my implementation, when a node is 'dead', it simply stops sending messages and stops sending responses to requests.

Hence, this test satisfies one part of **consideration 5**. This test effectively has the coordinator silently leaving the network, and this initialises one or more elections by nodes that detect this through the timeout. Further, this test validates **consideration 1** even more, as we have the detection of a failed coordinator and subsequent election.

To directly test this:
```bash
go test ./lib -v -run Test_BasicCrashAndReboot_DEMO
```

There will first be an election. After this election, the system will kill the coordinator. The subsequent re-election should pick the second-highest ID. Subsequently, the system will reboot the original coordinator, who should be re-elected.

### Durability Crash and Reboot Tests
This test kills off several nodes, ensuring that the correct coordinator is selected. This ensures the durability of the system despite multiple crashes and reboots.

Since we have several nodes that are **not** coordinators that are "silently leaving" the network, we fulfill the other part of **consideration (5.)** -- nodes that are not the coordinator that silently leave simply don't get updates, and can rejoin when necessary.

To directly test this:
```bash
go test ./lib -v -run Test_DurabilityCrashAndReboot_DEMO
```

This will kill off several nodes and report the new coordinator. Subsequently, it kills off 2 more, reporting the *new* coordinator. 

Finally, it reboots a higher ID node, which should become the newly-elected coordinator. Note that this uses a hardcoded 25 nodes, as dynamically adjusting the nodes to be killed based on `DEMO_NODECOUNT` was considered to be out of scope of the assignment.

### Non-Coordinator Crash During Election
This test kills off a non-coordinator node during the election, and reboots it later. The non-coordinator node should start an election, and proceed to get bullied into accepting the coordinator.

This satisfies **consideration 3b**, where a non-coordinator node fails during election.

To directly test this:
```bash
go test ./lib -v -run Test_NonCoordCrashDuringElection_DEMO
```

This will start an election. After that, the coordinator is killed. During the subsequent election, one of the non-coordinators is killed -- hence it won't receive the announcement message. Subsequently it is rebooted, and starts its own election before accepting 'defeat'.

### Partial Announcement Tests
Here, we simulate the case where a newly-elected coordinator fails before announcing to all nodes.

In this test, we simulate this scenario by first killing the nodes with the highest and second highest IDs.

We manually set the coordinators of the nodes:
- Half of the nodes will have their coordinator manually set to the second highest coordinator (simulating that the announcement message got through to them)
- The other half will have their coordinator manually set to the highest coordinator (simulating that they missed the announcement from the second-highest coordinator)

Finally, we initialise the system (having already set the coordinators). The nodes should detect the failed coordinators and adjust to the next best coordinator accordingly.

This test satisfies **consideration 3a**. The newly elected coordinator fails during the announcement stage such that some nodes receive the information about the new coordinator, but others did not.

Note that this simulated approach also applies for the case where a newly-awakened node that becomes the newly-elected coordinator fails during the announcement phase.

In both cases, the nodes in the system would have differing coordinator IDs. This is resolved after a new election initialised by any of the nodes that detect the failure of their coordinator.

To directly test this:
```bash
go test ./lib -v -run Test_PartialAnnouncement_DEMO
```

This will create the above simulated scenario, and result in the third-highest ID winning.

### Synchronisation, Crash and Reboot Tests
1. Start up $N$ nodes, and update the coordinator node with a new value. Ensure value is propagated through the system.
2. Shut down the coordinator. After the re-election, update the new coordinator with a new value. Ensure value is propagated through the system.
3. Reboot the original coordinator. After the re-election, update the new coordinator with a new value. Ensure value is propagated through the system.

This is a system test that fully tests our system's ability to satisfy consideration 1. Here, we show that our system supports:
- Joint Synchronisation: All live nodes will have the same value of the coordinator (at least immediately after the propagation of the update).
- Election: Even if the coordinator dies, there will be elections among the surviving nodes to select the next best coordinator.

We can be confident of the election process due to the Basic Crash and Reboot Tests.

To directly test this:
```bash
go test ./lib -v -run Test_SyncCrashReboot_DEMO
```

### Miscellaneous Tests
#### Best Case
The textbook best case for the Bully Algorithm re-election process is when the node with the next highest ID detects the crash of the coordinator. In such a case, the node only needs to send a self-election message to the coordinator, and proceed to declare its victory.

To simulate this case, I needed to disable the dead coordinator detection in all nodes but the second-highest ID.

To directly test this:
```bash
go test ./lib -v -run Test_SimulateBestCase_DEMO
```

Note that the actual "test" starts AFTER the initial election of the coordinator, hence you will have to wait until that initial election is done. There are system prompts to show *when* the actual test has startedwhich looks like this:
```
---SYSTEM: Election complete with coordinator N9.---

---SYSTEM: (Simulating best case, by disabling dead coordinator detection for all except second-highest ID node)---
2023/10/21 18:35:10 N0: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N1: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N7: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N6: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N2: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N3: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N4: Disabled detection of dead coordinator.
2023/10/21 18:35:10 N5: Disabled detection of dead coordinator.

---SYSTEM: Killing coordinator N9.---
2023/10/21 18:35:10 N9: Killed.
```

#### Worst Case

The textbook worst case for the Bully Algorithm is when the node with the lowest ID detects the crash of the coordinator. In such a case, the node broadcasts its self-election to all other nodes, triggering others to send their own vetoes and start their own elections.

To simulate this case, I disabled dead coordinator detection in all nodes but the lowest ID.

To directly test this:
```bash
go test ./lib -v -run Test_SimulateWorstCase_DEMO
```

Note that the actual "test" starts AFTER the initial election of the coordinator, hence you will have to wait until that initial election is done. The system prompt prior to the test looks like this:
```
---SYSTEM: Election complete with coordinator N9.---

---SYSTEM: (Simulating worst case, by disabling dead coordinator detection for all except lowest ID node)---
2023/10/21 18:32:51 N4: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N5: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N6: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N7: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N8: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N1: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N2: Disabled detection of dead coordinator.
2023/10/21 18:32:51 N3: Disabled detection of dead coordinator.

---SYSTEM: Killing coordinator N9.---
```
