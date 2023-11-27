package main

import (
	"errors"
	"fmt"
	"sync"
	"log"
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
	centralPort NodePort
	nodePort NodePort
	nodes map[NodeId]NodePort

	cache map[PageId](*CachedPage)

	incomingPageMsg Message // An incoming page from another node. This is all the node needs to block for, whether node is making a read or write request
	incomingPageMsgLock *sync.Mutex

	exit chan bool
}

func NewNode(nodeId NodeId, centralPort NodePort, nodePorts map[NodeId]NodePort, exit chan bool) *Node {
	myPort, ok := nodePorts[nodeId]
	if !ok {
		panic(fmt.Sprintf("Could not find node port for N%d.", nodeId))
	}
	
	return &Node{
		nodeId,
		centralPort, myPort, nodePorts,
		make(map[PageId](*CachedPage)),
		InvalidMessage(), &sync.Mutex{},
		exit,
	}
}

func (n *Node) ClientRead(pageId PageId) string {
	log.Printf("N%d: READ from P%d...", n.nodeId, pageId)
	n.ensureCachedPageExists(pageId)
	n.cache[pageId].pageLock.Lock() // Locking is done at this level, to ensure that reads and writes are done 'atomically', including any invalidation
	n.RequestRead(pageId)

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
	n.RequestWrite(pageId)

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

			//log.Printf("N%d: Received %v from N%d for P%d.", n.nodeId, GetMessageType(msg.MsgType), msg.SrcId, msg.PageId)

			switch msg.MsgType {
			case MSG_RP, MSG_WP, MSG_WI:
				n.incomingPageMsgLock.Lock()
				n.incomingPageMsg = msg
				n.incomingPageMsgLock.Unlock()
			case MSG_RF:
				n.recvReadForward(msg.SrcId, msg.PageId)
			case MSG_WF:
				n.recvWriteForward(msg.SrcId, msg.PageId)
			case MSG_IV:
				n.recvInvalidate(msg.SrcId, msg.PageId)
			default:
				panic(fmt.Sprintf("N%d: Received unexpected message %v from N%d", n.nodeId, GetMessageType(msg.MsgType), msg.SrcId))
			}
		case <-n.exit:
			log.Printf("N%d: Exiting.", n.nodeId)
			return
		}
	}
}

func (n *Node) recvReadForward(rdrId NodeId, pageId PageId) {
	n.ensureCachedPageExists(pageId)
	_, err := n.getCachedPage(pageId)
	if err != nil {
		panic(fmt.Sprintf("N%d: Received RF for invalid page P%d from N%d.", n.nodeId, pageId, rdrId))
	}

	// 1. Obtain local page lock
	n.cache[pageId].pageLock.Lock(); n.cache[pageId].pageLock.Unlock()

	// 2. Set page to READ_ONLY
	n.cache[pageId].access = ACCESS_READONLY

	// 3. Send RP to the reader
	n.sendToNode(MSG_RP, rdrId, pageId, n.cache[pageId].page.Data)

	log.Printf("N%d: P%d sent to N%d for READ.", n.nodeId, pageId, rdrId)
}

func (n *Node) recvWriteForward(wtrId NodeId, pageId PageId) {
	n.ensureCachedPageExists(pageId)
	_, err := n.getCachedPage(pageId)
	if err != nil {
		panic(fmt.Sprintf("N%d: Received WF for invalid page P%d from N%d.", n.nodeId, pageId, wtrId))
	}

	// 1. Obtain local page lock
	n.cache[pageId].pageLock.Lock(); n.cache[pageId].pageLock.Unlock()

	// 2. Set page to INVALID
	n.cache[pageId].access = ACCESS_INVALID

	// 3. Send WP to the reader
	n.sendToNode(MSG_WP, wtrId, pageId, n.cache[pageId].page.Data)

	log.Printf("N%d: P%d invalidated and sent to N%d for WRITE.", n.nodeId, pageId, wtrId)
}

func (n *Node) recvInvalidate(wtrId NodeId, pageId PageId) {
	n.ensureCachedPageExists(pageId)

	// 0. If the reason for invalidation is OUR own write, then the writing goroutine already holds the lock, AND has already invalidated the page. we can just immediately send an invalidation confirmation.
	if wtrId == n.nodeId {
		n.sendToManager(MSG_IC, pageId)
		log.Printf("N%d: P%d invalidated.", n.nodeId, pageId)
		return
	}

	// 1. Obtain local page lock
	n.cache[pageId].pageLock.Lock(); n.cache[pageId].pageLock.Unlock()

	// 2. Set page to INVALID
	n.cache[pageId].access = ACCESS_INVALID

	// 3. Send invalidate confirmation
	n.sendToManager(MSG_IC, pageId)

	log.Printf("N%d: P%d invalidated.", n.nodeId, pageId)
}

func (n *Node) RequestRead(pageId PageId){
	_, err := n.getCachedPage(pageId)
	if err == nil {
		// CACHE_HIT (getCachedPage checks the access, too)
		return
	}

	// CACHE_MISS
	// reset page
	n.incomingPageMsgLock.Lock()
	n.incomingPageMsg = InvalidMessage()
	n.incomingPageMsgLock.Unlock()
	
	// 1. Send RQ to manager
	n.sendToManager(MSG_RQ, pageId)

	// 2. Block until we receive the page
	page_rcvd := false
	page := InvalidPage()
	for !page_rcvd {
		n.incomingPageMsgLock.Lock()
		if n.incomingPageMsg.MsgType == MSG_RP {
			if n.incomingPageMsg.PageId == pageId {
				page_rcvd = true
				msg := n.incomingPageMsg
				page = NewPage(msg.PageId, msg.Data)
				n.incomingPageMsgLock.Unlock()
				break
			}
		}
		n.incomingPageMsgLock.Unlock()
	}

	// 3. Store page
	n.setCachedPage(pageId, page, ACCESS_READONLY)

	// 4. Send RC to manager
	n.sendToManager(MSG_RC, pageId)	
}

func (n *Node) RequestWrite(pageId PageId) {
	cachedPage, err := n.getCachedPage(pageId)
	if err == nil {
		if cachedPage.access == ACCESS_READWRITE {
			return
		}
	}

	// CACHE_MISS
	// reset page
	n.incomingPageMsgLock.Lock()
	n.incomingPageMsg = InvalidMessage()
	n.incomingPageMsgLock.Unlock()

	// 1. Send WQ to manager
	n.sendToManager(MSG_WQ, pageId)

	// 1.5. Invalidate own page, if it's not already invalid.
	n.cache[pageId].access = ACCESS_INVALID

	// 2. Block until we receive the page
	page_rcvd := false
	page := InvalidPage()
	for !page_rcvd {
		n.incomingPageMsgLock.Lock()
		if n.incomingPageMsg.MsgType == MSG_WP {
			if n.incomingPageMsg.PageId == pageId {
				page_rcvd = true
				msg := n.incomingPageMsg
				page = NewPage(msg.PageId, msg.Data)
				n.incomingPageMsgLock.Unlock()
				break
			}
		} else if n.incomingPageMsg.MsgType == MSG_WI {
			if n.incomingPageMsg.PageId == pageId {
				// Our written page is a NEW page
				page_rcvd = true
				msg := n.incomingPageMsg
				page = NewPage(msg.PageId, "")
				n.incomingPageMsgLock.Unlock()
				break
			} else {
				panic(fmt.Sprintf("N%d: Received MSG_WI for P%d, expected for P%d.", n.nodeId, n.incomingPageMsg.PageId, pageId))
			}
		}
		n.incomingPageMsgLock.Unlock()
	}

	// 3. Store page
	n.setCachedPage(pageId, page, ACCESS_READWRITE)

	// 4. Send WC to manager
	n.sendToManager(MSG_WC, pageId)
}

func (n *Node) sendToManager(msgType MsgType, pageId PageId) {
	if msgType != MSG_RQ && msgType != MSG_RC && msgType != MSG_WQ && msgType != MSG_WC && msgType != MSG_IC {
		panic(fmt.Sprintf("N%d: Unexpected message type %v when sending to manager", n.nodeId, GetMessageType(msgType)))
	}
	msg := NewMessage(msgType, n.nodeId, pageId)
	n.centralPort.RecvChan <- msg
	//log.Printf("N%d: Send %v for P%d to Manager.", n.nodeId, GetMessageType(msgType), pageId)
}

func (n *Node) sendToNode(msgType MsgType, dstId NodeId, pageId PageId, pageData string) {
	if msgType != MSG_RP && msgType != MSG_WP {
		panic(fmt.Sprintf("N%d: Unexpected message type %v when sending to another node", n.nodeId, GetMessageType(msgType)))
	}
	msg := NewMessageWithData(msgType, n.nodeId, pageId, pageData)
	n.nodes[dstId].RecvChan <- msg
	//log.Printf("N%d: Send %v for P%d to N%d.", n.nodeId, GetMessageType(msgType), pageId, dstId)
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
