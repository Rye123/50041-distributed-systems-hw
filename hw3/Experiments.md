# Experiments
Note that this document doesn't contain any implementation descriptions. Thsoe are contained in the respective READMEs.
- Experiment 1 uses tests in the `System_test.go` file in both implementations.
- All other experiments use tests in the `Performance_test.go` file in `IvyFaultTolerant`.

## Experiment 1: Performance of IvyBasic vs IvyFaultTolerant (No faults)
We can test this using the `TestMultiWrite` test case for both `IvyBasic` and `IvyFaultTolerant`.

This test:
1. Initialises a system of **20 clients**. 
2. Each client **simultaneously writes** to the **same page**. This is done with goroutines.
3. Then, sequentially, each client reads from the page. 
4. The test passes if the resultant data read from each client is the same.

We can directly compare the performance between IvyBasic and IvyFaultTolerant by seeing how long it takes for the test to pass. 

To run this test, run `go run . -v --run TestMultiWrite` in the desired directory (i.e. `./IvyBasic` or `./IvyFaultTolerant`). Note that since Go caches test results, `go clean -testcache` may be needed to clear cached results.

| Implementation                     | Trial 1 | Trial 2 | Trial 3 | Average |
| ---------------------------------- | ------- | ------- | ------- | ------- |
| `IvyBasic`                         | 0.306s  | 0.378s  | 0.330s  | 0.338s  |
| `IvyFaultTolerant` (Timeout 1s)    | 6.100s  | 7.100s  | 7.100s  | 6.770s  |
| `IvyFaultTolerant` (Timeout 0.1s)  | 1.100s  | 1.222s  | 1.120s  | 1.147s  | 
| `IvyFaultTolerant` (Timeout 0.01s) | 0.421s  | 0.486s  | 0.422s  | 0.443s  | 

We have the following observations:
- `IvyFaultTolerant`'s performance is **heavily dependent** on the timeout period set. This is the time a client waits to determine that the primary is down, and the time a CM waits to determine that they have won the election.
	- In the assumptions for `IvyFaultTolerant`, we assume that if a CM is alive, the round-trip time of a communication with the CM will be **bounded** by the timeout period.
	- Hence, a huge risk of lowering the timeout is violating that assumption -- for instance, if we drop the timeout too low, it is possible for both CMs to assume they have won the election before they receive an `ELECT_NO` (since they timed out when waiting for the `ELECT_NO`)
- `IvyFaultTolerant` is **consistently slower** than `IvyBasic`. 
	- This can be heavily attributed to the timeouts needed to determine a primary. Let the default timeout of the system be $T$. 
		 - At the start, with no primary, we first need a client to trigger an election (and hence the client needs to wait for $1 T$). 
		 - Then, both CMs need to wait for an `ELECT_NO` from the other. The winner of the election receives no `ELECT_NO`, but must first wait for $1 T$ before it can conclude that it has won the election.
	- Further, since `IvyFaultTolerant` relies on retransmission of requests, if it is currently handling a request, ==it will drop new requests==. Clients end up having to retransmit after waiting for the timeout period. Hence, a lot of the performance difference ends up being because of nodes waiting for timeouts, as indicated by the major difference when the timeout period is changed.
- Even after reducing timeouts to 10 milliseconds, we observe that there is still a distinctly poorer performance by `IvyFaultTolerant`.
	- This can be attributed to the additional messages that need to be sent by the primary to the backup as updates. 
	- Every RQ or WQ triggers a RQ_RECV or WQ_RECV message to be sent to the backup.
	- Every RC or WC triggers a RC_RECV or WC_RECV message to be sent to the backup.
	- This additional message complexity necessarily reduces the performance of `IvyFaultTolerant`.


## Experiment 2: Performance of IvyFaultTolerant (Single fault VS No faults)
In general, a single 'round' conducts a single write request and N read requests:
1. Has a node $i$ write data to a page
2. Has **all** nodes read data from that page. The test passes if the data is the expected data and is consistent.

This 'round' is repeated 6 times, each time varying the node to write to ($i$). Further, we set the timeout period to 1 second.

We run the following tests:
- `TestFaultless_PERF`: Runs the full test **without** any faults. 
- `TestSingleFault_PERF`: Primary is killed after the first three rounds. 
- `TestSingleFaultRandom_PERF`: Primary is killed after a random delay between 900 and 1100 milliseconds.
  - Note that in our scenario with a timeout of 1 second, it takes around 1000 milliseconds for the CMs to decide a winner. This range offers the best variety of potential outcomes (e.g. CM crash before election, CM crash during election, CM crash after election).
- `TestSingleFaultAndReboot_PERF`: Primary is killed after the first three rounds, and rebooted after the fourth.

This can be run in the `./IvyFaultTolerant` directory with:
```bash
go test . -v --run [Test Name]
```

| Implementation             | Trial 1 | Trial 2 | Trial 3 | Average |
| -------------------------- | ------- | ------- | ------- | ------- |
| `TestFaultless`            | 1.040s  | 1.040s  | 1.040s  | 1.040s  |
| `TestSingleFault`          | 3.060s  | 3.080s  | 3.050s  | 3.063s  |
| `TestSingleFaultRandom`    | 3.060s  | 3.070s  | 1.030s  | 2.387s  |
| `TestSingleFaultAndReboot` | 3.080s  | 3.050s  | 3.080s  | 3.070s  |

We have the following observations:
- A single fault heavily affects performance. This is likely due to the effect of the election. 
  - A client sending a request needs to first wait for the request to timeout, then needs to wait for the election to be concluded.
  - The election takes at least a single timeout period to conclude, 1 second in this case.
  - Hence, we can conclude that a single fault causes a delay of around two timeout periods (2 seconds in this case), which is reflected in the performance comparison table.
- The reboot of the dead former primary does not affect performance significantly.
  - In our implementation, the backup that takes over as primary **does not relinquish control** as long as it remains as the primary.
  - Hence, the only overhead is likely due to message traffic between the reawakened CM and the new primary CM. This could account for the slightly worse performance of the third test case.
- The random kill test case varies, based on the scenario that occurs:
  - If the primary crashes before or during election, then it is simply the `TestFaultless` scenario after the backup wins the election.
  - If the primary crashes after election (and winning), then it takes another 2 timeout periods (1 for a client to detect the failure, another for the backup to win) to proceed (hence the `TestSingleFault` scenario).
  
  
## Experiment 3: Performance of IvyFaultTolerant (Multiple faults of primary CM VS No faults)
(Note to marker): In the question, it says multiple faults of the primary. Say we have two CMs CM1 and CM2, where CM1 is the initial primary. I'm not sure if this question is asking for multiple faults of the **current** primary (i.e. this experiment, where the newly elected primary fails each round) or multiple faults of CM1.

Here, we compare the performance of the faultless scenario with the scenario where the **current primary** fails multiple times. For example, after the election of CM1 as primary, CM1 fails, causing CM2 to become the primary. CM1 reboots after this, and after some time the current primary CM2 fails, causing CM1 to become the primary again. The test elaborated later does this 'switching' of CMs to test the performance of the system in this state.

We use the same `TestFaultless_PERF` test above as a benchmark, with the following additional test.
- `TestMultiFaultsOfPrimary_PERF`: After each round, the **current primary** is killed, and rebooted after the next round. Essentially, we reboot a killed CM, and kill the current CM each round.

This can be run in the `./IvyFaultTolerant` directory with:
```bash
go test . -v --run [Test Name]
```


| Implementation             | Trial 1 | Trial 2 | Trial 3 | Average |
| -------------------------- | ------- | ------- | ------- | ------- |
| `TestFaultless`            | 01.050s | 01.050s | 01.040s | 01.046s |
| `TestMultiFaultsOfPrimary` | 11.710s | 11.270s | 11.210s | 11.397s | 

We have the following observations:
- In the face of multiple faults of the current primary, the performance is significantly hampered -- far more than in previous experiments.
  - As noted in previous experiments, it takes at least 2 timeout periods (1 for failure detection, 1 for winning an election) for the system to proceed.
  - Hence, if the primary has to switch every round, we expect a total of 5 'switches' in 6 rounds. This gives us a total of 10 timeout periods where the system is not doing anything but waiting for a timeout to occur.
  - Since our timeout period is 1 second, we can expect the system to have at least 10 seconds of time waiting for timeouts -- which explains the results where, on average, the scenario with multiple faults takes 10.351 seconds more than the faultless scenario.
  
  
## Experiment 4: Performance of IvyFaultTolerant (Multiple faults of primary and backup CM VS No faults)

Here, we compare the performance of the faultless scenario with the scenario where the the current primary and the current backup fails multiple times. Note that we assume that there is at least one CM active at all times -- either the backup or the primary is alive.

We use the same `TestFaultless_PERF` test above as a benchmark, with the following additional test.
- `TestMultiFaultsOfPrimaryAndBackup_PERF`: After each round, we reboot whichever node we've killed, and randomly kill a node. This node could either be a primary or a backup.

This can be run in the `./IvyFaultTolerant` directory with:
```bash
go test . -v --run [Test Name]
```


| Implementation             | Trial 1 | Trial 2 | Trial 3 | Trial 4 | Trial 5 | Trial 6 | Average |
| -------------------------- | ------- | ------- | ------- | ------- | ------- | ------- | ------- |
| `TestFaultless`            | 1.050s  | 1.050s  | 1.040s  | 1.040s  | 1.040s  | 1.050s  | 1.045s  |
| `TestMultiFaultsOfPrimary` | 7.168s  | 7.126s  | 3.102s  | 7.165s  | 3.135s  | 5.123s  | 5.470s  |

We have the following observations:
- The performance is on average, worse than the faultless run for similar reasons as Experiment 3: the loss of the primary causes a delay of at least 2 timeout periods.
- However, the crash of a backup has no significant effect on the performance. This explains the on-average better performance compared to Experiment 3, where the current primary is the one that is consistently failing each round.
