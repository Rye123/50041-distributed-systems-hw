# IvyFaultTolerant

A fault-tolerant version of Ivy. This involves two central managers that follow a primary-backup model.
- We have the following assumptions:
	1. We assume that during a crash, all data on a CM is lost. This necessitates that at least one CM is alive at a time in our system. 
	2. We use an assumption similar to that used for the Bully Algorithm. Messages must be delivered and responded to within a bounded time. If a CM is alive, it is **guaranteed** to receive a given message and respond within a given time, otherwise it is guaranteed to be a dead CM.
	3. Fault tolerance only applies to the central manager, not the hosts. If a host crashes, the system may be unable to progress.
- Further details about the actual implementation are in the [Design](./Design.md) page.

Timeouts are used to determine CM failure. To modify the timeout, modify `TIMEOUT_INTV` in `lib.go`. However, note that too low a value will be problematic due to Assumption (2).

## Usage
We have two separate files of test cases -- one is `System_test.go` to ensure there are no problems in a non-faulting system, another is a new `FT_test.go` file to test faulting scenarios. `System_test.go` was described in the [README for basic Ivy](../IvyBasic/README.md).

The new tests are described here:
- **Test Single Fault BEFORE Requests (`TestSingleFaultBeforeWrite`)**: This tests the ability for the backup to take over smoothly, before any state has been initialised in the system.
	1. Once the system is initialised, we kill the primary CM before any request occurs.
	2. We then do the standard usage tests on the backup CM to ensure that it can handle the necessary reads and writes.
- **Test Single Fault AFTER Requests (`TestSingleFaultAfterRequest`)**: This tests the ability for the backup to take over smoothly, after one or more writes have been completed to the primary.
	1. Once the system is initialised, we write to a single page, and read from it from all the nodes. This ensures that for that page, there is an owner and a copyset consisting of all the nodes.
	2. We then kill the primary CM. 
	3. We then write to the same page. This request should trigger an election before retransmission. Then, when reading from that same page from all the nodes, we should still have consistent data and the latest data.
	4. We repeat step (3.) from varying other nodes, ensuring that write privileges can be passed to and invalidated without any trouble.
- **Test Single Fault and Reboot (`TestSingleFaultAfterRequest`)**: This kills the primary. After an election, the original primary is rebooted, and the new primary is killed.
	1. Once the system is initialised, we write to a single page, and read from it from all the nodes. This ensures that for that page, there is an owner and a copyset consisting of all the nodes.
	2. We then kill the primary CM, CM-1. The backup CM, CM-2, should take over when the hosts find themselves unable to reach CM-1.
	3. We then write to the same page. This request should trigger an election before retransmission. Then, when reading from that same page from all the nodes, we should still have consistent data and the latest data.
	4. We reboot CM-1. Requests should still go to CM-2, as our system asserts that the primary refuses to relinquish control.
	5. We kill CM-2. Further requests should timeout, triggering an election that elects CM-1.
	- Note that all this while, we are writing to the same page and ensuring that all nodes, when reading data, read the same, latest written data. This ensures that the consistency of data throughout the system remains even as we mess with the CMs.
- **Test Multiple Faults (`TestMultiFaults`)**: This is the above test, but repeated multiple times (though there's always at least one primary online). In essence, we kill the primary, write and read from it, reboot the old primary and kill the new primary. We repeat this several times.

To run these tests:
```bash
go test . -v
```

## Deliverables
Refer to [Design](./Design.md) for the design, including a set of sequence diagrams that explore the potential issues of fault tolerance and how this design solves them. 

Refer to [Experiments](../Experiments.md) for the various experiments required for the assignment.
