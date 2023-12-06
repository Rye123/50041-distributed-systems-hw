package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	ACCESS_INVALID = 0
	ACCESS_READONLY = 1
	ACCESS_READWRITE = 2
)

type CachedPage struct {
	access int
	pageLock *sync.Mutex // locked while this is being read/written to
	page Page
}

func InvalidCachedPage() *CachedPage {
	return &CachedPage{0, &sync.Mutex{}, InvalidPage()}
}

type Node struct {
	nodeId NodeId
	
	cmId NodeId // Current CM ID
	cmIdLock *sync.Mutex
	
	cmPorts map[NodeId]NodePort // Communication with CMs
	nodePort NodePort
	nodes map[NodeId]NodePort // Communication with Nodes

	cache map[PageId](*CachedPage)

	incomingPageMsgChan chan Message // Channel for incoming pages from another node
	incomingPageMsgLock *sync.Mutex

	counter int
	counterLock *sync.Mutex

	requestState RequestState
	requestId string
	requestAckChan chan bool
	requestLock *sync.Mutex

	exit chan bool
}

func NewNode(nodeId NodeId, cmPorts map[NodeId]NodePort, nodePorts map[NodeId]NodePort, exit chan bool) *Node {
	if nodeId <= 0 {
		panic(fmt.Sprintf("Invalid Node ID %d.", nodeId))
	}
	
	myPort, ok := nodePorts[nodeId]
	if !ok {
		panic(fmt.Sprintf("Could not find node port for N%d.", nodeId))
	}
	
	return &Node{
		nodeId,
		InvalidNodeId, &sync.Mutex{}, cmPorts,
		myPort, nodePorts,
		make(map[PageId](*CachedPage)),
		make(chan Message), &sync.Mutex{},
		0, &sync.Mutex{},
		REQUEST_IDLE, "", make(chan bool), &sync.Mutex{},
		exit,
	}
}

func (n *Node) ClientRead(pageId PageId) string {
	log.Printf("N%d: READ from P%d...", n.nodeId, pageId)
	n.ensureCachedPageExists(pageId)
	n.cache[pageId].pageLock.Lock() // Locking is done at this level, to ensure that reads and writes are done 'atomically', including any invalidation

	reqId := n.generateMessageId()
	n.RequestRead(pageId, reqId)

	// Now, it's READ_ONLY
	data := n.cache[pageId].page.Data
	n.cache[pageId].pageLock.Unlock()
	log.Printf("N%d: READ from P%d COMPLETED: %v", n.nodeId, pageId, data)
	return data
}

func (n *Node) ClientWrite(pageId PageId, data string) {
	log.Printf("N%d: WRITE to P%d: %v...", n.nodeId, pageId, data)
	n.ensureCachedPageExists(pageId)
	n.cache[pageId].pageLock.Lock() // Locking is done at this level, to ensure that reads and writes are done 'atomically', including any invalidation
	
	reqId := n.generateMessageId()
	n.RequestWrite(pageId, reqId)

	// Now, it's READ_WRITE
	n.cache[pageId].page.Data = data
	n.cache[pageId].pageLock.Unlock()
	log.Printf("N%d: WRITE to P%d COMPLETED", n.nodeId, pageId)
}

func (n *Node) Listen() {
	//log.Printf("N%d: Listening.", n.nodeId)
	for {
		select {
		case msg, ok := <-n.nodePort.RecvChan:
			if !ok {
				log.Printf("N%d: Recv channel closed.", n.nodeId)
				return
			}

			//log.Printf("N%d: Received %v from N%d for P%d (ID: %v).", n.nodeId, GetMessageType(msg.MsgType), msg.SrcId, msg.PageId, msg.MsgId)

			switch msg.MsgType {
			// Standard IVY Messages: SrcID = Reader or Writer Node Id
			case MSG_RP, MSG_WP, MSG_WI:
				n.requestLock.Lock()
				if n.requestState == REQUEST_IDLE {
					panic(fmt.Sprintf("N%d: Received %v from N%d, but request state is idle", n.nodeId, GetMessageType(msg.MsgType), msg.SrcId))
				}
				n.requestLock.Unlock()
				n.incomingPageMsgChan <- msg
			case MSG_RF:
				// Received READ_FORWARD
				rdrId := msg.SrcId; pageId := msg.PageId; reqId := msg.MsgId
				n.ensureCachedPageExists(pageId)
				_, err := n.getCachedPage(pageId)
				if err != nil {
					panic(fmt.Sprintf("N%d: Received %v for invalid page P%d. (%v)", n.nodeId, GetMessageType(msg.MsgType), pageId, reqId))
				}

				n.cache[pageId].pageLock.Lock()
				n.cache[pageId].access = ACCESS_READONLY
				n.sendToNode(reqId, MSG_RP, rdrId, pageId, n.cache[pageId].page.Data)
				n.cache[pageId].pageLock.Unlock()
				log.Printf("N%d: P%d sent to N%d for READ", n.nodeId, pageId, rdrId)
			case MSG_WF:
				// Received WRITE_FORWARD
				wtrId := msg.SrcId; pageId := msg.PageId; reqId := msg.MsgId
				if wtrId == n.nodeId {
					// Requesting write for self
					// If WF to self, we can just send it straight to our incoming page msg chan
					n.incomingPageMsgChan <- NewMessageWithData(reqId, MSG_WP, n.nodeId, pageId, n.cache[pageId].page.Data)
					continue
				}
				
				n.ensureCachedPageExists(pageId)
				_, err := n.getCachedPage(pageId)
				if err != nil {
					panic(fmt.Sprintf("N%d: Received %v for invalid page P%d. (%v): %v", n.nodeId, GetMessageType(msg.MsgType), pageId, reqId, err))
				}

				n.cache[pageId].pageLock.Lock()
				n.cache[pageId].access = ACCESS_INVALID
				
				n.sendToNode(reqId, MSG_WP, wtrId, pageId, n.cache[pageId].page.Data)
				n.cache[pageId].pageLock.Unlock()
				log.Printf("N%d: P%d invalidated and sent to N%d for WRITE.", n.nodeId, pageId, wtrId)
			case MSG_IV:
				// Received INVALIDATION
				wtrId := msg.SrcId; pageId := msg.PageId; reqId := msg.MsgId
				n.ensureCachedPageExists(pageId)

				// If we're the writer, then the writer already owns the lock. Just acknowledge the invalidation.
				if wtrId == n.nodeId {
					n.cache[pageId].access = ACCESS_INVALID
					n.sendToManager(reqId, MSG_IC, pageId)
					log.Printf("N%d: P%d invalidated.", n.nodeId, pageId)
					continue
				}

				n.cache[pageId].pageLock.Lock()
				n.cache[pageId].access = ACCESS_INVALID
				n.sendToManager(reqId, MSG_IC, pageId)
				n.cache[pageId].pageLock.Unlock()
				log.Printf("N%d: P%d invalidated.", n.nodeId, pageId)
				
			case MSG_RC_ACK:
				n.requestLock.Lock()
				if msg.MsgId == n.requestId && n.requestState == REQUEST_RQ {
					n.requestId = ""
					n.requestState = REQUEST_IDLE
					n.requestAckChan <- true
				} else {
					if n.requestState != REQUEST_RQ {
						panic(fmt.Sprintf("N%d: Received unexpected RC_ACK (Current state is %d)", n.nodeId, n.requestState))
					}
					panic(fmt.Sprintf("N%d: Received RC_ACK for %v, expected %v", n.nodeId, msg.MsgId, n.requestId))
				}
				n.requestLock.Unlock()
			case MSG_WC_ACK:
				n.requestLock.Lock()
				if msg.MsgId == n.requestId && n.requestState == REQUEST_WQ {
					n.requestId = ""
					n.requestState = REQUEST_IDLE
					n.requestAckChan <- true
				} else {
					if n.requestState != REQUEST_WQ {
						panic(fmt.Sprintf("N%d: Received unexpected WC_ACK (Current state is %d)", n.nodeId, n.requestState))
					}
					panic(fmt.Sprintf("N%d: Received WC_ACK for %v, expected %v", n.nodeId, msg.MsgId, n.requestId))
				}
				n.requestLock.Unlock()

			// Election Win from CM: SrcId = CM ID
			case MSG_ELWIN:
				// Update CM ID
				n.cmIdLock.Lock()
				n.cmId = msg.SrcId
				n.cmIdLock.Unlock()
				//log.Printf("N%d: CM%d set as Central Manager.", n.nodeId, n.cmId)
			default:
				panic(fmt.Sprintf("N%d: Received unexpected message %v from N%d", n.nodeId, GetMessageType(msg.MsgType), msg.SrcId))
			}
			//log.Printf("N%d: Message handled (ID %v)", n.nodeId, msg.MsgId)
		case <-n.exit:
			log.Printf("N%d: Exiting.", n.nodeId)
			return
		}
	}
}

func (n *Node) RequestRead(pageId PageId, reqId string){
	_, err := n.getCachedPage(pageId)
	if err == nil {
		// CACHE_HIT (getCachedPage checks the access, too)
		return
	}

	// CACHE_MISS
	// reset page
	n.incomingPageMsgLock.Lock()
	n.incomingPageMsgChan = make(chan Message, 1)
	n.incomingPageMsgLock.Unlock()

	n.requestLock.Lock()
	n.requestState = REQUEST_RQ
	n.requestId = reqId
	n.requestAckChan = make(chan bool, 1)
	n.requestLock.Unlock()
	
	// 1. Send RQ to manager
	n.sendToManager(reqId, MSG_RQ, pageId)

	// 2. Block until we receive the page
	page := InvalidPage()
	timer := time.NewTimer(TIMEOUT_INTV)

	select {
	case recv_page_msg := <-n.incomingPageMsgChan:
		if recv_page_msg.MsgType == MSG_RP {
			timer.Stop()
			page = NewPage(recv_page_msg.PageId, recv_page_msg.Data)
		} else {
			panic(fmt.Sprintf("N%d: Got message type %v, expected MSG_RP", n.nodeId, GetMessageType(recv_page_msg.MsgType)))
		}
	case <-timer.C:
		// Timeout
		log.Printf("N%d: Timeout on request read.", n.nodeId)
		
		n.cmIdLock.Lock()
		n.cmId = InvalidNodeId
		n.cmIdLock.Unlock()

		n.StartElection() // This blocks until the election is done

		// Re-send ID to manager
		n.RequestRead(pageId, reqId)
		return	
	}

	if page.PageId == InvalidPageId {
		panic(fmt.Sprintf("N%d: Got Invalid Page", n.nodeId))
	}

	// 3. Store page
	n.setCachedPage(pageId, page, ACCESS_READONLY)
	n.requestReadConfirm(pageId, reqId)
}

func (n *Node) requestReadConfirm(pageId PageId, reqId string) {
	// 4. Send RC to manager
	n.sendToManager(reqId, MSG_RC, pageId)

	// 5. Block until RC_ACK received (i.e. requestState reset to REQUEST_IDLE)
	timer := time.NewTimer(TIMEOUT_INTV)

	select {
	case <-n.requestAckChan:
		timer.Stop()
	case <-timer.C:
		// Timeout
		log.Printf("N%d: Timeout on request read confirm.", n.nodeId)
		
		n.cmIdLock.Lock()
		n.cmId = InvalidNodeId
		n.cmIdLock.Unlock()

		// Reset request
		n.requestLock.Lock()
		n.requestState = REQUEST_IDLE
		n.requestLock.Unlock()

		n.StartElection() // This blocks until the election is done

		// Re-send ID to manager
		n.requestReadConfirm(pageId, reqId)
		return
	}

	// Request completed
}

func (n *Node) RequestWrite(pageId PageId, reqId string) {
	cachedPage, err := n.getCachedPage(pageId)
	if err == nil {
		if cachedPage.access == ACCESS_READWRITE {
			return
		}
	}

	// CACHE_MISS
	// reset page
	n.incomingPageMsgLock.Lock()
	n.incomingPageMsgChan = make(chan Message, 1)
	n.incomingPageMsgLock.Unlock()

	n.requestLock.Lock()
	n.requestState = REQUEST_WQ
	n.requestId = reqId
	n.requestAckChan = make(chan bool, 1)
	n.requestLock.Unlock()

	// 1. Send WQ to manager
	n.sendToManager(reqId, MSG_WQ, pageId)

	// 1.5. Invalidate own page, if it's not already invalid.
	n.cache[pageId].access = ACCESS_INVALID

	// 2. Block until we receive the page
	page := InvalidPage()
	timer := time.NewTimer(TIMEOUT_INTV)

	select {
	case recv_page_msg := <-n.incomingPageMsgChan:
		timer.Stop()
		if recv_page_msg.MsgType == MSG_WP {
			timer.Stop()
			if recv_page_msg.SrcId == n.nodeId {
				// We're the owner
				page = n.cache[pageId].page
			} else {
				page = NewPage(recv_page_msg.PageId, recv_page_msg.Data)
			}
		} else if recv_page_msg.MsgType == MSG_WI {
			timer.Stop()
			if recv_page_msg.PageId == pageId {
				// Our written page is a NEW page
				page = NewPage(recv_page_msg.PageId, "")
			} else {
				panic(fmt.Sprintf("N%d: Got MSG_WI for P%d, expected for P%d.", n.nodeId, recv_page_msg.PageId, pageId))
			}
		} else {
			panic(fmt.Sprintf("N%d: Got message type %v, expected MSG_WP", n.nodeId, GetMessageType(recv_page_msg.MsgType)))
		}
	case <-timer.C:
		// Timeout
		log.Printf("N%d: Timeout on request write.", n.nodeId)
		
		n.cmIdLock.Lock()
		n.cmId = InvalidNodeId
		n.cmIdLock.Unlock()

		// Reset request
		n.requestLock.Lock()
		n.requestState = REQUEST_IDLE
		n.requestLock.Unlock()

		n.StartElection() // This blocks until the election is done

		// Re-send ID to manager
		n.RequestWrite(pageId, reqId)
		return
	}

	if page.PageId == InvalidPageId {
		panic(fmt.Sprintf("N%d: Got Invalid Page", n.nodeId))
	}

	// 3. Store page
	n.setCachedPage(pageId, page, ACCESS_READWRITE)
	n.requestWriteConfirm(pageId, reqId)
}

func (n *Node) requestWriteConfirm(pageId PageId, reqId string) {
	// 4. Send WC to manager
	n.sendToManager(reqId, MSG_WC, pageId)

	// 5. Block until WC_ACK received (i.e. requestState reset to REQUEST_IDLE)
	timer := time.NewTimer(TIMEOUT_INTV)

	select {
	case <-n.requestAckChan:
		timer.Stop()
	case <-timer.C:
		// Timeout
		log.Printf("N%d: Timeout on request write confirm for %v.", n.nodeId, reqId)
		
		n.cmIdLock.Lock()
		n.cmId = InvalidNodeId
		n.cmIdLock.Unlock()

		n.StartElection() // This blocks until the election is done

		// Re-send ID to manager
		n.requestWriteConfirm(pageId, reqId)
		return
	}

	// Request completed
}

func (n *Node) sendToManager(msgId string, msgType MsgType, pageId PageId) {
	if msgType != MSG_RQ && msgType != MSG_RC && msgType != MSG_WQ && msgType != MSG_WC && msgType != MSG_IC {
		panic(fmt.Sprintf("N%d: Unexpected message type %v when sending to manager", n.nodeId, GetMessageType(msgType)))
	}
	msg := NewMessage(msgId, msgType, n.nodeId, pageId)

	// Identify port to send to
	cmId := n.cmId
	if cmId == InvalidNodeId {
		cmId = n.StartElection() // Block until we DO get a cmID
	}

	if cmId == InvalidNodeId {
		panic(fmt.Sprintf("N%d: No CM ID!", n.nodeId))
	}

	n.cmPorts[cmId].RecvChan <- msg
	//log.Printf("N%d: Send %v for P%d to Manager.", n.nodeId, GetMessageType(msgType), pageId)
}

func (n *Node) sendToNode(msgId string, msgType MsgType, dstId NodeId, pageId PageId, pageData string) {
	if msgType != MSG_RP && msgType != MSG_WP {
		panic(fmt.Sprintf("N%d: Unexpected message type %v when sending to another node", n.nodeId, GetMessageType(msgType)))
	}
	msg := NewMessageWithData(msgId, msgType, n.nodeId, pageId, pageData)
	n.nodes[dstId].RecvChan <- msg
}

func (n *Node) StartElection() NodeId {
	n.cmIdLock.Lock()
	cmId := n.cmId
	n.cmIdLock.Unlock()
	if cmId == InvalidNodeId {
		log.Printf("N%d: Start election.", n.nodeId)
		// Start an election
		el_msg := NewMessage(n.generateMessageId(), MSG_EL, n.nodeId, InvalidPageId)
		for _, cmPort := range n.cmPorts {
			cmPort.RecvChan <- el_msg
		}

		// Block until cmId is set i.e. once one of the CMs broadcasts an election win, doesn't have to be the election we just started
		for cmId == InvalidNodeId {
			n.cmIdLock.Lock()
			cmId = n.cmId
			n.cmIdLock.Unlock()
		}
	}
	//log.Printf("N%d: Election completed, detected CM%d.", n.nodeId, n.cmId)
	return cmId
}

func (n *Node) generateMessageId() string {
	n.counterLock.Lock(); defer n.counterLock.Unlock()
	msgId := fmt.Sprintf("MSG_N%d_%d", n.nodeId, n.counter)
	n.counter++
	return msgId
}

func (n *Node) ensureCachedPageExists(pageId PageId) {
	if _, ok := n.cache[pageId]; !ok {
		n.cache[pageId] = &CachedPage{ ACCESS_INVALID, &sync.Mutex{}, InvalidPage() }
	}
}

func (n *Node) getCachedPage(pageId PageId) (*CachedPage, error) {
	cached_page := n.cache[pageId]
	if cached_page.access == ACCESS_INVALID {
		return cached_page, errors.New("Page is invalidated.")
	}
	return cached_page, nil
}

func (n *Node) setCachedPage(pageId PageId, page Page, access int) {
	if pageId != page.PageId { panic(fmt.Sprintf("Tried to set cached page where pageId = %d but page.PageId = %d", pageId, page.PageId)) }

	n.cache[pageId].access = access
	n.cache[pageId].page = page
}
