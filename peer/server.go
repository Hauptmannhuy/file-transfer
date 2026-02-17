package server

import (
	"encoding/json"
	"errors"
	"file-transfer/bridge"
	"file-transfer/logger"
	scaner "file-transfer/scan"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"time"
)

const listenerPort uint16 = 6071
const requestPort uint16 = 6070

type clientNetMsgType uint8

const (
	fileSend clientNetMsgType = iota + 1
	p2pRequest
)

type clientNetMsg struct {
	MsgType clientNetMsgType
	Data    []byte
}

type peerConn struct {
	identifier string
	conn       *net.TCPConn
	channel    chan clientNetMsg
}

type eventHandler func(request bridge.BridgeMessage, response bridge.BridgeMessage) error
type eventHandlerMap map[bridge.ClientCmdEnum]eventHandler

type P2PMessageType string

const (
	MsgHandshake P2PMessageType = "handshake"
	MsgOk        P2PMessageType = "ok"
	msgRefused   P2PMessageType = "refused"
	MsgError     P2PMessageType = "error"
)

type p2pMessage struct {
	Type P2PMessageType `json:"type"`
}

type P2Pserver struct {
	connections   map[string]peerConn
	waitableConn  map[string]peerConn
	eventHandlers eventHandlerMap
	memoryBridge  bridge.Bridge
}

type errHandshakeRefused string

func (err errHandshakeRefused) String() string {
	return string(err)
}

func InitServer(bridge bridge.Bridge) P2Pserver {
	server := P2Pserver{
		connections:  map[string]peerConn{},
		waitableConn: map[string]peerConn{},
		memoryBridge: bridge,
	}
	server.eventHandlers = attachEventHandlers(server)
	return server
}

func (server P2Pserver) ProccessQueue() {
	for msg := range server.memoryBridge.Get() {
		logger.Log.Info("received message from memory bridge to server endpoint with payload", "msg_peer", msg.PeerID, "msg_request_p2p", msg.RequestP2P, "msg_send_file_data", msg.SendFileData, "msg_err", msg.Err)
		response := bridge.BridgeMessage{}
		handler := server.eventHandlers[msg.Type]
		if err := handler(*msg, response); err != nil {
			response.Err = err
		}

		server.memoryBridge.Send(&response)
	}
}

func attachEventHandlers(server P2Pserver) eventHandlerMap {
	eventHandlers := eventHandlerMap{}
	eventHandlers[bridge.CmdRequestP2P] = server.handleConnectToPeer
	return eventHandlers
}

func (server P2Pserver) handleSendFile(request bridge.BridgeMessage, response bridge.BridgeMessage) error {
	peerID := request.PeerID

	data, err := json.Marshal(request.SendFileData)
	if err != nil {
		return err
	}

	peer := server.connections[*peerID]
	clientNetMsg := clientNetMsg{
		MsgType: fileSend,
		Data:    data,
	}
	peer.channel <- clientNetMsg
	return nil
}

func (server P2Pserver) handleConnectToPeer(request bridge.BridgeMessage, response bridge.BridgeMessage) error {
	var err error
	ip := *request.RequestP2P
	if err = server.ConnectToPeer(ip); err != nil {
		logger.Log.Error("error in connect to peer handler", "error", err.Error())
	} else {
		logger.Log.Info("handler connect to peer succesfully finished")
	}
	return err
}

func (server P2Pserver) AcceptP2P(ip string) {
	server.memoryBridge.Send(&bridge.BridgeMessage{
		Type:       bridge.AcceptP2P,
		RequestP2P: &ip,
	})
}
func (server P2Pserver) Listen() {
	go server.ProccessQueue()

	listener, err := initListener()
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.AcceptTCP()
		logger.Log.Info("Incoming P2P request from", "ip", conn.RemoteAddr().String())
		if err != nil {
			logger.Log.Error("error accept tcp", "err", err.Error())
		}

		ipAddr := conn.RemoteAddr().String()
		peer := newPeerConn(ipAddr, conn)
		server.waitableConn[ipAddr] = peer
		server.AcceptP2P(ipAddr)
	}
}

func initListener() (*net.TCPListener, error) {
	localAddr := scaner.GetLocalHostAddr()
	ipAddr, err := netip.ParseAddr(localAddr.IP.String())
	if err != nil {
		return nil, err
	}
	laddr := net.TCPAddrFromAddrPort(netip.AddrPortFrom(ipAddr, listenerPort))
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return nil, err
	}

	logger.Log.Info("listener initialized on", "ip", localAddr.IP.String())
	return listener, nil
}

func (server P2Pserver) ConnectToPeer(ipStr string) error {
	logger.Log.Info(ipStr)
	var ip net.IP = net.ParseIP(ipStr)
	var raddr *net.TCPAddr
	var laddr *net.TCPAddr
	var err error
	listenerPort := strconv.Itoa(int(listenerPort))
	// requestPort := strconv.Itoa(int(requestPort))

	raddr, err = net.ResolveTCPAddr("tcp", ip.String()+":"+listenerPort)
	if err != nil {
		logger.Log.Error("error resolving raddr", "ip", ip.String(), "error", err.Error())
		return err
	}

	// laddr, err = net.ResolveTCPAddr("tcp", scaner.GetLocalHostAddr().IP.String())
	// if err != nil {
	// 	logger.Log.Error("error resolving laddr", "ip", ip.String(), "error", err.Error())
	// 	return err
	// }
	fmt.Printf("requesting from %s to %s\n", laddr.String(), raddr.String())
	conn, err := net.DialTCP("tcp", nil, raddr)
	var peer peerConn
	var netError net.Error
	if err == nil {
		logger.Log.Info("dial tcp is successfull..?")
		err := server.requestHandshake(conn)
		if err != nil {
			logger.Log.Error(err.Error())
			if err := conn.Close(); err != nil {
				logger.Log.Error(err.Error())
			}
			return err
		}

		peer = newPeerConn(ip.String(), conn)
		runSession(peer)
		server.connections[ip.String()] = peer
	} else {
		if errors.As(err, &netError) {
			if netError.Timeout() {
				logger.Log.Info("error dial tcp is occuring most likely due to firewall blocking port, check your settings and try again", "port", listenerPort)
			}
		}
	}
	logger.Log.Error(err.Error())
	return err
}

func (server P2Pserver) requestHandshake(conn *net.TCPConn) error {
	var buffer []byte = make([]byte, 64)
	var netError net.Error
	var response p2pMessage

	msg := p2pMessage{
		Type: MsgHandshake,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err = conn.Write(bytes); err != nil {
		return err
	}

	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 15)); err != nil {
		return err
	}

	_, err = conn.Read(buffer)
	if err != nil && errors.As(err, &netError) {
		logger.Log.Error("error during conn read", "error", err)
		if netError.Timeout() {
			return fmt.Errorf("timeout during p2p request handshake")
		}
	}

	if err = json.Unmarshal(buffer, &response); err != nil {
		return err
	}

	switch response.Type {
	case MsgOk:
		return nil
	case msgRefused:
		return errors.New("peer refused connection")
	}

	return fmt.Errorf("unknown error, response message has undefined type %s", response.Type)
}

func (server P2Pserver) SignalPeer(message bridge.BridgeMessage) error {
	log.Fatalf("%s", "not implemented")
	return nil
}

func newPeerConn(ip string, conn *net.TCPConn) peerConn {
	peer := peerConn{
		identifier: ip,
		conn:       conn,
		channel:    make(chan clientNetMsg),
	}
	return peer
}

func runSession(peer peerConn) {
	conn := peer.conn
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
				var incomingMsg clientNetMsg
				err := json.Unmarshal(buffer, &incomingMsg)
				if err != nil {
					logger.Log.Error("error decoding p2p message", "err", err.Error())
				}
				logger.Log.Info("recevied message from peer")
			}
		}
	}()

	go func() {
		for msg := range peer.channel {

			data, err := json.Marshal(msg)
			if err != nil {
				log.Println("error encoding message send to peer")
			}
			_, err = peer.conn.Write(data)
			if err != nil {
				log.Println("error sending message to peer")
			}

		}
	}()

}
