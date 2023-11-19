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

### Voting Protocol
The prior implementations heavily relied on the fact that "votes" (i.e. request acknowledgements) could not be rescinded. Hence, it was safe to handle each incoming message concurrently regardless of the message type, relying on mutex locks to ensure safety.

In the Voting Protocol (with deadlock prevention), we needed to be able to rescind a vote, which meant that if the above implementation of concurrent message handling were to be naively used, there would be separate goroutines handling incoming requests and releases. This made it far more difficult as any mutex lock over the node's vote would need to be *shared* among goroutines, destroying the purpose of the mutex lock in the first place.

Hence, I had a separate goroutine that handled requests and releases sequentially (with a channel). A modified version of the local priority queue was used to keep track of the requests that had yet to be voted for. 

To account for late messages (e.g. late `VOTE`s and late `RESCIND`s), each 'election' (where a node requested to enter the CS) had a unique election ID. 
- Further, since a node could receive a `RESCIND` before the actual `VOTE`, a `voteSession` manager was used to keep track of the actual vote status of nodes.

## Testing
Our main test involves simultaneously creating 100 nodes. Then, we initialise 100 goroutines for each node to sequentially enter and exit the critical section.

## Printing the Table
Running `main.go` prints the timing calculations:
```bash
go run .
```


On one run with 100 nodes and 100 milliseconds spent for each node in the critical section, we got the following output:
```
---LAMPORT'S SHARED PRIORITY QUEUE---
Lamport's Shared Priority Queue:
        System Time: 25.5614355s
        Node Times: map[0:11.4506768s 1:24.7807231s 2:22.8611053s 3:11.6612147s 4:11.9007673s 5:12.1622319s 6:18.1890863s 7:18.5875113s 8:5.1825406s 9:18.7000105s 10:12.4633053s 11:21.6601635s 12:20.7575138s 13:5.7213705s 14:12.6368385s 15:24.8925054s 16:18.860356s 17:21.7918766s 18:25.0035107s 19:18.4359698s 20:19.1464117s 21:12.8421099s 22:6.2775348s 23:22.991368s 24:21.9017158s 25:13.1133352s 26:25.1148774s 27:22.0936041s 28:20.9193658s 29:13.3972105s 30:25.5590862s 31:13.592105s 32:19.3775238s 33:13.867219s 34:23.1519805s 35:6.5015505s 36:14.103993s 37:14.3722046s 38:21.059553s 39:6.7945516s 40:14.6635273s 41:23.3373009s 42:24.113763s 43:14.9343827s 44:7.1289054s 45:23.4479549s 46:25.2267472s 47:19.517087s 48:7.3631043s 49:25.337519s 50:23.5585683s 51:7.7175178s 52:8.1181113s 53:22.2267528s 54:8.3872022s 55:24.4476299s 56:15.1439984s 57:15.3777153s 58:8.6380474s 59:15.5360375s 60:9.1608053s 61:15.8470331s 62:9.4239392s 63:9.6856544s 64:16.0642635s 65:16.3335193s 66:19.6858333s 67:22.3505996s 68:16.5200071s 69:21.1674788s 70:24.5583802s 71:23.6695252s 72:24.6698873s 73:24.2248975s 74:20.0032259s 75:22.4950063s 76:25.4483532s 77:23.7803886s 78:20.1473208s 79:21.3375571s 80:23.8920276s 81:16.7772033s 82:20.3407976s 83:16.952611s 84:24.3364771s 85:17.1596759s 86:17.3699357s 87:9.9407746s 88:21.5039813s 89:20.4656511s 90:17.6200958s 91:24.0039542s 92:17.867039s 93:10.1639642s 94:20.6029413s 95:10.3810894s 96:10.5783251s 97:10.9905864s 98:11.2287663s 99:22.6901891s]

---RICART AND AGRAWALA'S OPTIMISATION---
Ricart and Agrawala's Optimisation:
        System Time: 20.1466057s
        Node Times: map[0:5.1162846s 1:5.2602431s 2:17.8306523s 3:5.2917752s 4:4.5101193s 5:4.6606753s 6:5.0113887s 7:4.881089s 8:5.1202149s 9:17.9369757s 10:5.3168664s 11:5.3484215s 12:20.0362127s 13:19.7034997s 14:5.4987278s 15:19.8146143s 16:5.6743243s 17:5.8081926s 18:6.2819126s 19:20.1466057s 20:6.7857799s 21:6.9602002s 22:18.0462414s 23:18.1567752s 24:7.1659274s 25:7.4586837s 26:7.5829198s 27:7.8346694s 28:7.9912098s 29:8.1827151s 30:8.3329096s 31:8.660479s 32:8.8840823s 33:9.0728298s 34:9.4213735s 35:9.8632898s 36:10.0370876s 37:10.2429999s 38:10.4011667s 39:18.2676822s 40:10.6235796s 41:18.3775768s 42:10.8138851s 43:10.9500379s 44:11.1951504s 45:11.4952053s 46:19.9253374s 47:11.6299267s 48:18.4887206s 49:11.7929441s 50:12.0060304s 51:12.1709457s 52:12.3352733s 53:18.5992772s 54:18.7101595s 55:12.509877s 56:18.8217993s 57:12.670054s 58:12.8129194s 59:13.0469525s 60:13.1888364s 61:13.3051486s 62:13.528186s 63:13.7438195s 64:13.866706s 65:14.0439774s 66:18.9323409s 67:19.0431722s 68:14.2080178s 69:14.3323573s 70:14.5681315s 71:14.7263084s 72:14.8898186s 73:15.0281076s 74:15.1542184s 75:15.2561875s 76:15.3817994s 77:15.522157s 78:15.6462502s 79:19.1537796s 80:15.7900007s 81:15.9162746s 82:19.2628817s 83:16.1374573s 84:16.2496912s 85:16.4045706s 86:16.5279479s 87:16.648664s 88:16.7910137s 89:16.9033795s 90:17.0364809s 91:19.3722131s 92:19.4816809s 93:19.5930995s 94:17.15788s 95:17.2851801s 96:17.3966047s 97:17.5066763s 98:17.6180624s 99:17.7302346s]

---VOTING PROTOCOL---
Voting Protocol:
        System Time: 11.1028647s
        Node Times: map[0:10.5351042s 1:131.9416ms 2:10.6461913s 3:252.4831ms 4:9.3140326s 5:5.8640673s 6:8.6521136s 7:7.7666368s 8:4.3311131s 9:352.3285ms 10:8.4319148s 11:462.4623ms 12:9.647148s 13:5.9749883s 14:573.114ms 15:7.878077s 16:685.4049ms 17:798.634ms 18:4.4405535s 19:6.9901359s 20:7.9890034s 21:908.6066ms 22:1.0185505s 23:8.1105988s 24:1.129168s 25:7.1011148s 26:10.0906682s 27:6.086349s 28:8.8720791s 29:9.7583727s 30:9.5364933s 31:9.8703869s 32:10.7581642s 33:6.2080088s 34:1.2387446s 35:4.548933s 36:1.3597004s 37:1.4698091s 38:4.6578223s 39:1.5711089s 40:1.6922717s 41:4.7658034s 42:8.9822402s 43:4.8852541s 44:4.9778375s 45:1.8028988s 46:5.0879334s 47:1.9036445s 48:10.2027863s 49:2.014176s 50:6.3087848s 51:2.1335892s 52:6.4197512s 53:11.0928429s 54:7.2129209s 55:5.1987444s 56:9.4255835s 57:10.8687184s 58:9.9802538s 59:6.5319836s 60:10.9809627s 61:8.2107028s 62:9.0927669s 63:5.3200313s 64:10.3138142s 65:8.7612112s 66:2.2434497s 67:8.5419711s 68:2.3431139s 69:2.4537729s 70:5.4205326s 71:2.5744508s 72:6.642481s 73:5.5315917s 74:5.6425272s 75:2.6844313s 76:6.7697495s 77:2.7955184s 78:6.8795804s 79:7.3238004s 80:2.8958227s 81:7.4339159s 82:9.2034971s 83:3.0166629s 84:3.1266929s 85:3.2275241s 86:3.3379742s 87:7.5446312s 88:3.4582628s 89:3.5592509s 90:3.6692314s 91:3.7902824s 92:3.9004919s 93:7.6556755s 94:4.0108473s 95:5.7524655s 96:4.1110952s 97:4.2216903s 98:8.3216917s 99:10.4247304s]
```
