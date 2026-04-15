package protocol

import (
	"errors"
	"net"
)

type ClientNetMsgType uint8

const (
	fileSend ClientNetMsgType = iota + 1
	p2pRequest
)

type ClientNetMsg struct {
	MsgType ClientNetMsgType
	Data    []byte
}

type P2PMessageType string
type P2PMessageAction string

const (
	MsgHandshake P2PMessageType = "handshake"
)

const (
	MsgOk         P2PMessageAction = "ok"
	MsgRefused    P2PMessageAction = "refused"
	MsgError      P2PMessageAction = "error"
	MsgRequestP2P P2PMessageAction = "request"
)

type P2PMessage struct {
	Type   P2PMessageType   `json:"type"`
	Action P2PMessageAction `json:"action"`
}

type PeerDesicion int

const (
	PeerAccepted = iota + 1
	PeerDenied
)

type PeerConn struct {
	Identifier string
	Conn       *net.TCPConn
	Channel    chan ClientNetMsg
}

func NewPeerConn(ip string, conn *net.TCPConn) PeerConn {
	peer := PeerConn{
		Identifier: ip,
		Conn:       conn,
		Channel:    make(chan ClientNetMsg),
	}
	return peer
}

var intToP2PActionMap = map[int]P2PMessageAction{
	PeerAccepted: MsgOk, PeerDenied: MsgRefused,
}

func IntToP2PAction(response int) (P2PMessageAction, error) {
	action, ok := intToP2PActionMap[response]
	if !ok {
		return "", errors.New("couldn't translate int response to p2p action")
	}
	return action, nil
}
