package main

import "fmt"

type MsgType int
type NodeId int
type PageId int

const (
	MSG_RQ MsgType = iota // Read Request
	MSG_RF // Read Forward
	MSG_RC // Read Confirm
	MSG_RP // Read Page: Where owner pushes page with data to the reader
	MSG_WQ // Write Request
	MSG_WF // Write Forward
	MSG_WC // Write Confirm
	MSG_WI // Write Init: Where server tells node this is a new page
	MSG_WP // Write Page: Where owner pushes page with data to writer
	MSG_IV // Invalidate
	MSG_IC // Invalidate Confirm
	MSG_INVALID
)
const InvalidNodeId = NodeId(-1)
const InvalidPageId = PageId(-1)

func GetMessageType(msgType MsgType) string {
	switch msgType {
	case MSG_RQ:
		return "MSG_RQ"
	case MSG_RF:
		return "MSG_RF"
	case MSG_RC:
		return "MSG_RC"
	case MSG_RP:
		return "MSG_RP"
	case MSG_WQ:
		return "MSG_WQ"
	case MSG_WF:
		return "MSG_WF"
	case MSG_WC:
		return "MSG_WC"
	case MSG_WI:
		return "MSG_WI"
	case MSG_WP:
		return "MSG_WP"
	case MSG_IV:
		return "MSG_IV"
	case MSG_IC:
		return "MSG_IC"
	}
	return "INVALID"
}

type Message struct {
	MsgType MsgType
	SrcId NodeId
	PageId PageId
	Data string    // Blank for all requests except for RP
}

func NewMessage(msgType MsgType, srcId NodeId, pageId PageId) Message {
	return Message{msgType, srcId, pageId, ""}
}

func NewMessageWithData(msgType MsgType, srcId NodeId, pageId PageId, data string) Message {
	if msgType != MSG_RP && msgType != MSG_WP {
		panic(fmt.Sprintf("Created unexpected message type %v with data %v", GetMessageType(msgType), data))
	}
	return Message{msgType, srcId, pageId, data}
}

func InvalidMessage() Message {
	return Message{MSG_INVALID, InvalidNodeId, InvalidPageId, ""}
}

type Page struct {
	PageId PageId
	Data string
}

func NewPage(pageId PageId, data string) Page {
	return Page{pageId, data}
}

func InvalidPage() Page {
	return Page{InvalidPageId, ""}
}

// Communication channel for a node -- other nodes push into RecvChan to send a message
type NodePort struct {
	NodeId NodeId
	RecvChan chan Message
}
