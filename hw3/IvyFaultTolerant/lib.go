package main

import (
	"fmt"
	"time"
)

type MsgType int
type NodeId int
type PageId int

const (
	MSG_RQ MsgType = iota // Read Request
	MSG_RF // Read Forward
	MSG_RC // Read Confirm
	MSG_RC_ACK // Read Confirmation Acknowledgement: CM ack of the RC
	MSG_RP // Read Page: Where owner pushes page with data to the reader
	MSG_WQ // Write Request
	MSG_WF // Write Forward
	MSG_WC // Write Confirm
	MSG_WC_ACK // Write Confirmation Acknowledgement: CM ack of the WC
	MSG_WI // Write Init: Where server tells node this is a new page
	MSG_WP // Write Page: Where owner pushes page with data to writer
	MSG_IV // Invalidate
	MSG_IC // Invalidate Confirm
	MSG_EL // Election Start, to CM
	MSG_ELWIN // Election Win, from CM
	MSG_CM_UPDATE
	MSG_CM_RQ_RECV
	MSG_CM_RC_RECV
	MSG_CM_WQ_RECV
	MSG_CM_WC_RECV
	MSG_CM_ELECT_ME
	MSG_CM_ELECT_NO
	MSG_INVALID
)
const InvalidNodeId = NodeId(0)
const InvalidPageId = PageId(-1)

func GetMessageType(msgType MsgType) string {
	switch msgType {
	case MSG_RQ:
		return "MSG_RQ"
	case MSG_RF:
		return "MSG_RF"
	case MSG_RC:
		return "MSG_RC"
	case MSG_RC_ACK:
		return "MSG_RC_ACK"
	case MSG_RP:
		return "MSG_RP"
	case MSG_WQ:
		return "MSG_WQ"
	case MSG_WF:
		return "MSG_WF"
	case MSG_WC:
		return "MSG_WC"
	case MSG_WC_ACK:
		return "MSG_WC_ACK"
	case MSG_WI:
		return "MSG_WI"
	case MSG_WP:
		return "MSG_WP"
	case MSG_IV:
		return "MSG_IV"
	case MSG_IC:
		return "MSG_IC"
	case MSG_EL:
		return "MSG_EL"
	case MSG_ELWIN:
		return "MSG_ELWIN"
	case MSG_CM_UPDATE:
		return "MSG_CM_UPDATE"
	case MSG_CM_RQ_RECV:
		return "MSG_CM_RQ_RECV"
	case MSG_CM_RC_RECV:
		return "MSG_CM_RC_RECV"
	case MSG_CM_WQ_RECV:
		return "MSG_CM_WQ_RECV"
	case MSG_CM_WC_RECV:
		return "MSG_CM_WC_RECV"
	case MSG_CM_ELECT_ME:
		return "MSG_CM_ELECT_ME"
	case MSG_CM_ELECT_NO:
		return "MSG_CM_ELECT_NO"
	}
	return "INVALID"
}

type Message struct{
	MsgId string
	MsgType MsgType
	SrcId NodeId
	PageId PageId
	Data string    // Blank for all requests except for RP
}

func NewMessage(msgId string, msgType MsgType, srcId NodeId, pageId PageId) Message {
	if msgType >= MSG_CM_UPDATE {
		panic(fmt.Sprintf("Invalid msgType for Message: %v", GetMessageType(msgType)))
	}
	return Message{msgId, msgType, srcId, pageId, ""}
}

func NewMessageWithData(msgId string, msgType MsgType, srcId NodeId, pageId PageId, data string) Message {
	if msgType >= MSG_CM_UPDATE {
		panic(fmt.Sprintf("Invalid msgType for Message: %v", GetMessageType(msgType)))
	}
	if msgType != MSG_RP && msgType != MSG_WP {
		panic(fmt.Sprintf("Created unexpected message type %v with data %v", GetMessageType(msgType), data))
	}
	return Message{msgId, msgType, srcId, pageId, data}
}

func InvalidMessage() Message {
	return Message{"", MSG_INVALID, InvalidNodeId, InvalidPageId, ""}
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

type InternalPort struct {
	NodeId NodeId
	RecvChan chan CMMessage
}

// Internal message exchanged between CMs
type CMMessage struct {
	MsgId string // original ID if forwarded, election ID if not
	MsgType MsgType
	SrcId NodeId // original node ID if forwarded, CM ID if not
	
	PageId PageId // pageId for the forwarded message, set for all forwarded messages

	// Page Information -- only shared for RQ and WQ messages, empty otherwise
	SrcCopysets map[PageId]([]NodeId)
	SrcOwners   map[PageId]NodeId
}

func NewCMElectionMessage(electionId string, msgType MsgType, srcCmId NodeId) CMMessage {
	if msgType < MSG_CM_ELECT_ME {
		panic(fmt.Sprintf("Invalid msgType for CM Election Message: %v", GetMessageType(msgType)))
	}

	return CMMessage{electionId, msgType, srcCmId, InvalidPageId, make(map[PageId]([]NodeId)), make(map[PageId]NodeId)}
}

// Internal message to send an update (i.e. a forwarded message)
func NewCMUpdate(reqId string, msgType MsgType, nodeId NodeId, pageId PageId, srcCopysets map[PageId]([]NodeId), srcOwners map[PageId]NodeId) CMMessage {
	if msgType < MSG_CM_UPDATE || msgType >= MSG_CM_ELECT_ME {
		panic(fmt.Sprintf("Invalid msgType for CM Update: %v", GetMessageType(msgType)))
	}
	
	return CMMessage{reqId, msgType, nodeId, pageId, srcCopysets, srcOwners}
}

// Current request state of a node or CM.
type RequestState int
const (
	REQUEST_IDLE RequestState = 0 // No request being handled
	REQUEST_RQ = 1 // Read Request being handled.
	REQUEST_WQ = 2 // Write Request being handled.
)


const TIMEOUT_INTV = time.Second * 1
