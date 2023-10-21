# PSET1 Q1 Part 2
## Problem
Consider some client-server architecture as follows. Several clients are registered to the server. Periodically, each client sends message to the server. Upon receiving a message, the server flips a coin and decides to either forward the message to all other registered clients (excluding the original sender of the message) or drops the message altogether.

Use Lamport’s logical clock to determine a total order of all the messages received at all the registered clients. Subsequently, present (i.e., print) this order for all registered clients to know the order in which the messages should be read.

## Solution
### Usage

This command assumes you are in the same directory as this `README.md` file (`hw1/q1/part1`).

```bash
go run main.go
```

**To safely exit, press `ENTER`**. This would also trigger every registered client to print out the **total order of the received messages**.
- In other words, every client should have the same order of messages (excluding the messages they themselves sent).
- This total order is determined by the logical clock timestamp of each message and the source ID of each message.
- If `CTRL-C` is used, this may not actually print out the total order.

### Organisation
The code is largely the same as in Part 1.

This time however, we implement Lamport's logical clock.
- To determine a **total order** among messages, we use the logical clock time. To deconflict two messages with the same logical clock value, the source ID of the message is used -- the message with a lower source ID is considered to be 'earlier' than a message with a higher source ID.
- The `Message` `struct` includes a timestamp recording the logical clock time at which the message was sent.
- Both `Server` and `Client` maintain their own logical clocks.
- In the `Send()` and `Handle()` methods of `Server` and `Client`, we modify the clock values as necessary.
  - Note that a broadcast by the server (used when it forwards the message) is considered as $N$ separate messages if it is broadcasting to $N$ clients.

