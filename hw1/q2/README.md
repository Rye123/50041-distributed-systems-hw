# PSET1 Q2

The bulk of the code is in the `lib` package, which implements the Bully algorithm among $N$ nodes that synchronise a string value using Go channels.
- Each node runs concurrently, using goroutines.
- Synchronisation is done using buffered channels.
  - Each node has a data channel and control message channel that it constantly reads from.
  - Any incoming message from another node is sent to these channels.
  
## Usage
TODO

## Considerations
The various considerations given in the question are resolved using Go tests.

### Basic Initialisation Tests
1. Start up $N$ nodes, wait for election to complete.
2. Ensure coordinator node is the node with the highest ID, $N-1$.

This satisfies consideration (4.). Since our implementation has every new node start its own election by default, we have multiple goroutines starting the election process simultaneously, and we check if this results in the highest ID becoming the coordinator regardless of the number of nodes starting an election.

### Basic Synchronisation Tests
1. Start up $N$ nodes, wait for election to complete.
2. Update coordinator node $N-1$ with a new value, and wait for value to be propagated (i.e. RTT + default timeout)
3. Ensure that ALL nodes have that modified value.

### Basic Crash and Reboot Tests
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
