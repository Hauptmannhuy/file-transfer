package events

import (
	"file-transfer/logger"
	"net"
)

type EventMsg struct {
	Type ClientCmdEnum
	Err  error

	IP             string `json:"ip"`
	StatusResponse bool   `json:"status_response"`

	LocalIP      string
	SendFileData SendFileData
	LocalNet     []string
	RawConn      *net.TCPConn
}

type SendFileData struct {
	Data     []byte
	FileName string
}

type ClientCmdEnum uint8

const (
	CmdRequestAddresses ClientCmdEnum = iota + 1
	CmdIdentifyHost
	CmdSendFile
	RequestAcceptP2P
	ResponseAcceptP2P
)

var clientCmdNames = map[ClientCmdEnum]string{
	CmdRequestAddresses: "RequestAddresses",
	CmdIdentifyHost:     "IdentifyHost",
	CmdSendFile:         "SendFile",
	ResponseAcceptP2P:   "ResponseP2P",
	RequestAcceptP2P:    "RequestAcceptP2P",
}

var eventTypeToProtocolPacket = map[ClientCmdEnum]func(msg *EventMsg) any{
	ResponseAcceptP2P: func(msg *EventMsg) any {
		return ResponseP2PRequest{
			IP:             msg.IP,
			StatusResponse: msg.StatusResponse,
		}
	},
	RequestAcceptP2P: func(msg *EventMsg) any {
		return AcceptP2PRequest{
			IP: msg.IP,
		}
	},
	CmdRequestAddresses: func(msg *EventMsg) any {
		return LocalNetAddresses{
			Addresses: msg.LocalNet,
		}
	},
	CmdIdentifyHost: func(msg *EventMsg) any {
		return IdentifyHostMemoryPacket{
			IP: msg.LocalIP,
		}
	},
}

type MemoryProtocolPacket struct {
	Error string `json:"error"`
	Data  any    `json:"data"`
}

func EventToMemoryProtocolPacket(msg *EventMsg) *MemoryProtocolPacket {
	var result *MemoryProtocolPacket = &MemoryProtocolPacket{}
	build, ok := eventTypeToProtocolPacket[msg.Type]
	if ok {
		result.Data = build(msg)

	}
	if msg.Err != nil {
		result.Error = msg.Err.Error()
	}

	logger.Log.Info("packed protocol packet", "packet", result)
	return result
}

type AcceptP2PRequest struct {
	IP string `json:"ip"`
}

type ResponseP2PRequest struct {
	IP             string `json:"ip"`
	StatusResponse bool   `json:"status_response"`
}

type LocalNetAddresses struct {
	Addresses []string `json:"local_net_addrs"`
}

type IdentifyHostMemoryPacket struct {
	IP string `json:"ip"`
}

func (c ClientCmdEnum) String() string {
	name, ok := clientCmdNames[c]
	if !ok {
		return "unknown command"
	}
	return name
}

func (c ClientCmdEnum) Exists() bool {
	_, ok := clientCmdNames[c]
	return ok
}
