# PSET1 Q1 Part 3
## Problem
Consider some client-server architecture as follows. Several clients are registered to the server. Periodically, each client sends message to the server. Upon receiving a message, the server flips a coin and decides to either forward the message to all other registered clients (excluding the original sender of the message) or drops the message altogether.

Use Vector clock to redo the assignment. Implement the detection of causality violation and print any such detected causality violation

## Solution
### Usage
```bash
go run main.go
```

To safely exit, press `ENTER`. As with part 2, this would also trigger every registered client to print out the **total order of the received messages**.
- In other words, every client should have the same order of messages (excluding the messages they themselves sent).
- This total order is determined by the vector clock timestamp of each message.

### Organisation
The code is largely the same as in Part 2.

This time however, we implement a vector clock.
- To determine a **total order** among messages, we use the vector clock time.
- Further, the `lib/Clock.go` file now extends the `ClockVal` type (previously an integer) to a `struct` with a `map` that maps a node ID to its clock value. The file also provides the necessary methods to operate on this type.

To simulate a causality violation, a client creates two messages with two timestamps $T$ and $(T+x)$ (where $x > 0$). The client sends the message with timestamp $(T+x)$ before the message with timestamp $T$, and this should be reported as a causality violation on the server's end.
- This has a 50% chance of occurring by default, this value can be modified in `main.go` under `CAUSALITY_VIOLATION_CHANCE`.
- For easy viewing, the client will log that it is deliberately causing a causality violation, and the server would log that a given message is a causality violation. 
- The default action is for the server to drop the message -- typically this means accepting the first message that comes, and dropping the second message that causes the causality violation. 
