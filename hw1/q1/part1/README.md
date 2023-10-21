# PSET1 Q1 Part 1
## Problem
Consider some client-server architecture as follows. Several clients are registered to the server. Periodically, each client sends message to the server. Upon receiving a message, the server flips a coin and decides to either forward the message to all other registered clients (excluding the original sender of the message) or drops the message altogether.

Simulate the behaviour of both server and clients via Goroutines.

## Solution
### Usage
This command assumes you are in the same directory as this `README.md` file (`hw1/q1/part1`).

```bash
go run main.go
```

To safely exit, press `ENTER`.

### Organisation

The code itself is split into a `lib` directory and a main file. 

In the main file (`./main.go`), there are several constants that can be modified to test the code:
- `CLIENT_COUNT`: Controls the number of clients (excluding the server)
- `SERVER_DROP_CHANCE`: Controls the probability of the server dropping a given message.
- `CLIENT_DELAY_FLOOR`/`CLIENT_DELAY_CEIL`: Sets the minimum and maximum delay of the clients' send interval.

When started, the `main` function initialises 1 server and `CLIENT_COUNT` number of clients in separate goroutines. Each client sends a message every few seconds (controlled by a random send interval in the range [`CLIENT_DELAY_FLOOR`, `CLIENT_DELAY_CEIL`]), and the server has a `SERVER_DROP_CHANCE` probability to drop the message, or forward it to every client except for the sender.
- Every send or receive message will be logged for easy tracing.

The `lib` directory contains the code needed for the client-server protocol. `Client.go` contains the code for each client, `Server.go` contains the code for the server, and `Message.go` gives the `struct` for the messages being exchanged between client and server.
