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

const defaultPort uint16 = 6070
const connectPort uint16 = 6071

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
	eventHandlers eventHandlerMap
	memoryBridge  bridge.Bridge
}

func InitServer(bridge bridge.Bridge) P2Pserver {
	server := P2Pserver{
		connections:  map[string]peerConn{},
		memoryBridge: bridge,
	}
	server.eventHandlers = attachEventHandlers(server)
	return server
}

func (server P2Pserver) ProccessQueue() {
	for message := range server.memoryBridge.Get() {
		log.Println("received message from memory part")
		response := bridge.BridgeMessage{}
		handler := server.eventHandlers[message.Type]
		if err := handler(*message, response); err != nil {
			response.Err = err
			logger.Log.Error(err.Error())
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
	ip := *request.RequestP2P
	return server.ConnectToPeer(ip)
}
func (server P2Pserver) Listen() {
	go server.ProccessQueue()

	listener, err := initListener()
	if err != nil {
		log.Fatal(err)
	}

	for {
		var incomingPeer string
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}

		ipAddr := conn.RemoteAddr().String()
		peer := newPeerConn(ipAddr, conn)
		runSession(peer)
		fmt.Printf("request from %s\n", incomingPeer)
	}
}

func initListener() (*net.TCPListener, error) {
	localAddr := scaner.GetLocalHostAddr()
	ipAddr, err := netip.ParseAddr(localAddr.IP.String())
	if err != nil {
		return nil, err
	}
	laddr := net.TCPAddrFromAddrPort(netip.AddrPortFrom(ipAddr, defaultPort))
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return nil, err
	}

	fmt.Printf("listening on %s\n", laddr.String())
	return listener, nil
}

func (server P2Pserver) ConnectToPeer(ipStr string) error {
	var ip net.IP = net.ParseIP(ipStr)
	var raddr *net.TCPAddr
	var laddr *net.TCPAddr
	var err error

	port := strconv.Itoa(int(defaultPort))

	raddr, err = net.ResolveTCPAddr("tcp", ip.String()+":"+port)
	if err != nil {
		return err
	}

	laddr, err = net.ResolveTCPAddr("tcp", scaner.GetLocalHostAddr().IP.String()+":"+strconv.Itoa(int(connectPort)))
	if err != nil {
		return err
	}
	fmt.Printf("requesting from %s to %s\n", laddr.String(), raddr.String())
	conn, err := net.DialTCP("tcp", laddr, raddr)
	var peer peerConn
	if err == nil {
		ok, err := server.requestHandshake(conn)
		if err != nil {
			return err
		} else if !ok {
			return errors.New("request handshake is failed due to unknown reason")
		}

		peer = newPeerConn(ip.String(), conn)
		runSession(peer)
		server.connections[ip.String()] = peer
	}
	return err
}

func (server P2Pserver) requestHandshake(conn *net.TCPConn) (bool, error) {
	var buffer []byte = make([]byte, 64)
	var netError net.Error
	var response p2pMessage

	msg := p2pMessage{
		Type: MsgHandshake,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return false, err
	}

	if _, err = conn.Write(bytes); err != nil {
		return false, err
	}

	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 30)); err != nil {
		return false, err
	}

	_, err = conn.Read(buffer)
	if err != nil && errors.As(err, &netError) {
		if netError.Timeout() {
			return false, fmt.Errorf("timeout during p2p request handshake")
		}
	}

	if err = json.Unmarshal(buffer, &response); err != nil {
		return false, err
	}

	switch response.Type {
	case MsgOk:
		return true, nil
	case msgRefused:
		return false, errors.New("peer refused connection")
	}

	return false, fmt.Errorf("unknown error, response message has undefined type %s", response.Type)
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
			fmt.Println("readed ", n, "bytes")
			fmt.Println("parsing...")
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
