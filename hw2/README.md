# HW2

We follow the following scenario: We have a **shared** memory space in the form of a struct `nodetypes.SharedMemory`.
- This struct provides the functions `EnterCS` and `ExitCS`, which modify the shared memory space.
- A simple implementation can be seen in `nodetypes.NaiveNode`, which simply acquires and releases the lock ignoring everyone else.

The `Orchestrator` allows us a convenient interface to simulate various test cases. By using it along with the `testing` package, we can create a readable and non-verbose series of tests.
- An example is in `Orchestrator_test.go`, which tests the orchestrator using the above naive implementation.
