# PSET1 Q2

The bulk of the code is in the `lib` package, which implements the Bully algorithm among $N$ nodes that synchronise a string value using Go channels.
- Each node runs concurrently, using goroutines.
- Synchronisation is done using buffered channels.
  - Each node has a data channel and control message channel that it constantly reads from.
  - Any incoming message from another node is sent to these channels.
  
## Usage

For manual testing, please use `main.go`.

However, the `lib/Node_test.go` file already provides most tests necessary to demonstrate handling of the cases mentioned in the question. Refer to Considerations for details.

To run the tests:
```bash
cd ./lib
go test -v
```

These tests could take up to 5 minutes to complete due to the number of system tests being used. To manually run a test:
```bash
cd ./lib
go test -v -run [test name]
```
- `[test name]` would be replaced by one of the tests under Considerations (e.g. `Test_BasicInit_5Nodes`).


## Considerations
The various considerations given in the question are resolved using Go tests.

I use an `Orchestrator` class in `lib`, which automatically creates the necessary nodes and provides helper functions to manipulate the system.

### Basic Initialisation Tests (`Test_BasicInit_[N]Nodes`, `N` = `[1, 5, 10, 50]`)
1. Start up $N$ nodes, wait for election to complete.
2. Ensure coordinator node is the node with the highest ID, $N-1$.

This satisfies consideration (4.). Since our implementation has every new node start its own election by default, we have multiple goroutines starting the election process simultaneously, and we check if this results in the highest ID becoming the coordinator regardless of the number of nodes starting an election.

### Complex Initialisation Test (`Test_ComplexInit`)
Here, we manually modify multiple nodes to consider themselves coordinators. Ideally, the system should fix itself once these coordinators start broadcasting -- higher ID nodes should attempt to elect themselves upon receiving a data broadcast from lower ID nodes.

### Basic Synchronisation Tests (`Test_BasicSync_[N]Nodes`, `N` = `[5, 25]`)
1. Start up $N$ nodes, wait for election to complete.
2. Update coordinator node $N-1$ with a new value, and wait for value to be propagated (i.e. RTT + default timeout)
3. Ensure that ALL nodes have that modified value.

We also have another test which replaces multiple nodes' values (simulating data corruption), and ensuring that the coordinator's value is the one that replaces all other values. This test is done in `Test_BasicSync_25Nodes_Corrupted`.

### Basic Crash and Reboot Tests (`Test_BasicCrashAndReboot_[N]Nodes`, `N` = `[5, 25]`)
1. Start up $N$ nodes, wait for election to complete.
2. Shut down coordinator node, wait for re-election to complete.
3. Ensure that the node with the next highest ID is selected (i.e. $N-2$).
4. Reboot original coordinator, wait for re-election to complete.
5. Ensure that node with the highest ID is selected (i.e. $N-1$)

### Durability Crash and Reboot Tests
1. Start up $N$ nodes, wait for election to complete.
2. Kill a random number of nodes.
3. Ensure that the node with the next highest ID is selected.
4. Reboot a random number of nodes, wait for re-election to complete.
5. Ensure next highest ID is selected.
6. Repeat step (4.) several times.

### Advanced Crash Tests
1. Start up $N$ nodes, wait for election to complete.
2. Shut down coordinator node.
3. Before new coordinator announces to all nodes, new coordinator crashes. This is implemented with a custom function.

### Synchronisation, Crash and Reboot Tests
1. Start up $N$ nodes, and update the coordinator node with a new value. Ensure value is propagated through the system.
2. Shut down the coordinator. After the re-election, update the new coordinator with a new value. Ensure value is propagated through the system.
3. Reboot the original coordinator. After the re-election, update the new coordinator with a new value. Ensure value is propagated through the system.

This satisfies consideration (1.). If this test succeeds, then we can be sure that we have:
- Joint synchronisation: All live nodes will have the same value as the coordinator.
- Election: Even if the coordinator dies, there will be an election to select the next best coordinator.

We can be confident of the election process due to the Basic Crash and Reboot Tests.
