package core

import (
	"encoding/json"
	"file-transfer/events"
	"file-transfer/logger"
	"file-transfer/protocol"
	scaner "file-transfer/scan"
	"fmt"
	"log"
	"os"
)

type Dispatcher struct {
	guiIn              chan (*events.EventMsg)
	guiOut             chan (*events.EventMsg)
	networkIn          chan (*events.EventMsg)
	networkOut         chan (*events.EventMsg)
	connections        map[string]protocol.PeerConn
	pendingConnections map[string]protocol.PeerConn
}

func InitializeDispatcher(guiIn chan (*events.EventMsg),
	guiOut chan (*events.EventMsg),
	networkIn chan (*events.EventMsg),
	networkOut chan (*events.EventMsg)) *Dispatcher {

	dispatcher := &Dispatcher{
		connections:        map[string]protocol.PeerConn{},
		pendingConnections: map[string]protocol.PeerConn{},

		guiIn:      guiIn,
		guiOut:     guiOut,
		networkIn:  networkIn,
		networkOut: networkOut,
	}
	return dispatcher
}

func (dispatcher *Dispatcher) Dispatch() {
	for {
		select {
		case msg := <-dispatcher.guiOut:
			dispatcher.dispatchIPCmsg(msg)
		case msg := <-dispatcher.networkOut:
			dispatcher.dispatchNetworkMsg(msg)
		}
	}
}

func (dispatcher *Dispatcher) dispatchNetworkMsg(msg *events.EventMsg) {
	var ingoingMsg events.EventMsg
	switch msg.Type {
	case events.RequestAcceptP2P:

		peer, exist := dispatcher.connections[msg.IP]
		if exist {
			logger.Log.Info("peer already connected", "addr", peer)
			return
		}

		dispatcher.pendingConnections[msg.IP] = protocol.NewPeerConn(msg.IP, ingoingMsg.RawConn)
		dispatcher.guiIn <- msg
	}
}

func (dispatcher *Dispatcher) dispatchIPCmsg(msg *events.EventMsg) {
	logger.Log.Info("dispatching IPC message with", "type", msg.Type, "payload", *msg)
	switch msg.Type {
	case events.CmdRequestAddresses:

		go func() {
			result := scaner.Scan()
			msg.LocalNet = result
			dispatcher.guiIn <- msg
		}()

	case events.RequestAcceptP2P:
		fmt.Println("before")
		dispatcher.networkIn <- msg
		fmt.Println("after")
	case events.CmdIdentifyHost:
		// ingoingMsg =
		dispatcher.guiIn <- &events.EventMsg{
			LocalIP: scaner.GetLocalHostAddr().IP.String(),
		}
	case events.ResponseAcceptP2P:
		dispatcher.networkIn <- msg
	case events.CmdSendFile:
		msg = &events.EventMsg{
			SendFileData: events.SendFileData{
				Data:     []byte{},
				FileName: "name",
			},
		}
	}
}

func runSession(peer protocol.PeerConn) {
	conn := peer.Conn
	conn.SetKeepAlive(true)
	var buffer []byte
	go func() {
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
			if n > 0 {
				var incomingMsg protocol.ClientNetMsg
				err := json.Unmarshal(buffer, &incomingMsg)
				if err != nil {
					logger.Log.Error("error decoding p2p message", "err", err.Error())
				}
				logger.Log.Info("recevied message from peer")
			}
		}
	}()

	go func() {
		for msg := range peer.Channel {

			data, err := json.Marshal(msg)
			if err != nil {
				log.Println("error encoding message send to peer")
			}
			_, err = peer.Conn.Write(data)
			if err != nil {
				log.Println("error sending message to peer")
			}

		}
	}()

}
