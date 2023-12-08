# IvyBasic

A basic implementation of Ivy.

## Usage
The test cases provided in `System_test.go` run the following tests on ten nodes:
- **Standard Usage (`TestStandardUsage`)**: 
	1. Write data to a single page to a single node.
	2. Read that page from every node in the system. All nodes should read the same data.
	3. Write data to the same page, from a different node.
	4. Read that page from every node in the system. All nodes should read the latest data.
- **Multi Write (`TestMultiWrite`)**: For fun, this test uses 20 nodes instead of 10.
	1. Using goroutines, we have all 20 nodes attempt to write data to the same page.
	2. Since we cannot accurately predict which node's write would be final, we simply check if all nodes, when reading the page, report the same data.

To run these tests, run:
```bash
go test . -v
```


## Implementation
In addition to the protocol described in class, my implementation includes additional functionality for registering a new page.

In the standard protocol, upon receiving a write request (WQ), the central manager (CM) sends a write forward (WF) message to the actual owner of the page.
- However, we don't have a protocol for when this page is **new**, and hence unknown to the CM.
- Hence, upon receiving a WQ for an **unknown page**, the CM responds with a write init (WI) message to the writer and registers the page.

### Node
Has an external interface `ClientRead(pageId) -> Data` and `ClientWrite(pageId, data)`. This in turn calls the necessary functions to communicate with other nodes or the central manager.
- Mutex locks are used to prevent possible problems such as when an invalidation comes in during a `ClientRead`.
