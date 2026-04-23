package server

import (
	"encoding/json"
	"errors"
	"file-transfer/events"
	"file-transfer/logger"
	"file-transfer/protocol"
	scaner "file-transfer/scan"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strconv"
	"time"
)

const listenerPort uint16 = 6071
const requestPort uint16 = 6070

type errHandshakeRefused string

func (err errHandshakeRefused) String() string {
	return string(err)
}

type P2Pserver struct {
	ingoing  chan (*events.EventMsg)
	outgoing chan (*events.EventMsg)
}

func InitServer(ingoing chan (*events.EventMsg), outgoing chan (*events.EventMsg)) P2Pserver {
	server := P2Pserver{
		ingoing:  ingoing,
		outgoing: outgoing,
	}
	return server
}

func (server P2Pserver) Listen() {

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

		server.outgoing <- &events.EventMsg{
			Type:    events.RequestAcceptP2P,
			IP:      conn.RemoteAddr().String(),
			RawConn: conn,
		}
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

func (server P2Pserver) ConnectToPeer(ipStr string) (*net.TCPConn, error) {
	logger.Log.Info(ipStr)
	var ip net.IP = net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("couldn't parse string %s into ip", ipStr)
	}

	var err error
	listenerPort := strconv.Itoa(int(listenerPort))

	if err != nil {
		logger.Log.Error("error resolving raddr", "ip", ip.String(), "error", err.Error())
		return nil, err
	}

	// conn, err := net.DialTCP("tcp", nil, raddr)
	d := net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := d.Dial("tcp", fmt.Sprintf("%s:%s", ipStr, listenerPort))
	// d.DialUDP()
	var netError net.Error
	if err == nil {
		logger.Log.Info("dial tcp is successfull..?")
		err := server.RequestHandshake(nil)
		if err != nil {
			logger.Log.Error(err.Error())
			if err := conn.Close(); err != nil {
				logger.Log.Error(err.Error())
			}
			return nil, err
		}

	} else {
		if errors.As(err, &netError) {
			if netError.Timeout() {
				logger.Log.Info("error dial tcp is occuring most likely due to firewall blocking port, check your settings and try again", "port", listenerPort)
			}
		}
	}
	logger.Log.Error(err.Error())
	return nil, err
}
func (server P2Pserver) ProccessQueue() {
	for msg := range server.ingoing {
		fmt.Println("here")
		logger.Log.Info("p2p server received message with type", "type", "data", msg)
		switch msg.Type {
		case events.RequestAcceptP2P:
			go func() {
				conn, err := server.ConnectToPeer(msg.IP)
				if err != nil {
					logger.Log.Error(err.Error())
				}

				response := events.EventMsg{
					Type:    events.ResponseAcceptP2P,
					Err:     err,
					RawConn: conn,
				}
				server.outgoing <- &response
			}()
		}

	}
}

func (server P2Pserver) RequestHandshake(conn *net.TCPConn) error {
	var buffer []byte = make([]byte, 64)
	var netError net.Error
	var response protocol.P2PMessage

	msg := protocol.P2PMessage{
		Type:   protocol.MsgHandshake,
		Action: protocol.MsgRequestP2P,
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

	switch response.Action {
	case protocol.MsgOk:
		return nil
	case protocol.MsgRefused:
		return errors.New("peer refused connection")
	}

	return fmt.Errorf("unknown error, response message has undefined type %s", response.Type)
}

func (server P2Pserver) SignalPeer(message events.EventMsg) error {
	log.Fatalf("%s", "not implemented")
	return nil
}
