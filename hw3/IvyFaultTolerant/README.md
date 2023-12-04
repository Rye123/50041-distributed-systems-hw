# IvyBasic

A basic implementation of Ivy.

## Implementation
### Central Manager
Handles read and write requests **sequentially**. Invalidation confirmations are handled concurrently.

### Node
Has an external interface `ClientRead(pageId) -> Data` and `ClientWrite(pageId, data)`. This in turn calls the necessary functions to communicate with other nodes or the central manager.
- Mutex locks are used to prevent possible problems such as when an invalidation comes in during a `ClientRead`.
