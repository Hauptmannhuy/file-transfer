package server

import (
	"file-transfer/ipc"
	"file-transfer/peers"
	scaner "file-transfer/scan"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"sync"
)

const defaultPort uint16 = 6070
const connectPort uint16 = 6071

type peerConn struct {
	identifier string
	conn       *net.TCPConn
	channel    chan string
}

type ConnQueue struct {
	queue []*net.TCPConn
	mu    *sync.Mutex
}

type hub struct {
}

func Start(ipcState *ipc.IPCstate) {

	// if len(scaner.HandshakedIPs) == 0 {
	// 	fmt.Println("There is no saved hosts that you can connect to. \n Performing search...")
	// 	peerConns := initList()
	// 	renderList(peerConns)
	// }

	go listen(ipcState)

}

func listen(ipc *ipc.IPCstate) {
	listener, err := initListener()
	if err != nil {
		log.Fatal(err)
	}

	for {
		var savedPeer *peers.SavedPeer
		var incomingPeer string
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}

		ipAddr := conn.RemoteAddr().String()
		peer := newPeerConn(ipAddr, conn)
		runSession(peer, ipc)
		fmt.Printf("request from %s\n", incomingPeer)
		fmt.Println(savedPeer)

		// respond with handshake if want to connect
		// save addr locally and named it
		// start exchange
	}
}

func PromtSaveRequest(ip string) *peers.SavedPeer {
	var peer *peers.SavedPeer
	peers.RegisterPeer(ip, "test peer")
	return peer
}

func initListener() (*net.TCPListener, error) {
	localAddr := scaner.GetLocalHostAddress()
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

func connect(ipStr string) (*net.TCPConn, error) {
	var ip net.IP = net.ParseIP(ipStr)
	var raddr *net.TCPAddr
	var laddr *net.TCPAddr
	var err error

	port := strconv.Itoa(int(defaultPort))

	raddr, err = net.ResolveTCPAddr("tcp", ip.String()+":"+port)
	if err != nil {
		return nil, err
	}

	// net.TCPAddrFromAddrPort()

	laddr, err = net.ResolveTCPAddr("tcp", scaner.GetLocalHostAddress().IP.String()+":"+strconv.Itoa(int(connectPort)))
	if err != nil {
		return nil, err
	}
	fmt.Printf("requesting from %s to %s\n", laddr.String(), raddr.String())
	return net.DialTCP("tcp", laddr, raddr)
}

func newPeerConn(ip string, conn *net.TCPConn) peerConn {
	peer := peerConn{
		identifier: ip,
		conn:       conn,
		channel:    make(chan string),
	}
	return peer
}

func runSession(peer peerConn, ipc *ipc.IPCstate) {
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
}
