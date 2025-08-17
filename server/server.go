package server

import (
	"file-transfer/ipc"
	"file-transfer/peers"
	scaner "file-transfer/scan"
	"file-transfer/tcp"
	"fmt"
	"log"
	"net"
	"sync"
)

type nodeCommand struct {
	ip   string
	conn *net.TCPConn
}

type ConnQueue struct {
	queue []*net.TCPConn
	mu    *sync.Mutex
}

func Start(ipcState *ipc.IPCstate) {

	// if len(scaner.HandshakedIPs) == 0 {
	// 	fmt.Println("There is no saved hosts that you can connect to. \n Performing search...")
	// 	connections := initList()
	// 	renderList(connections)
	// }

	go func() {
		for {
			// ipc.CheckRequestCommands(ipcState)
		}
	}()

	go wait()

}

func initList() []nodeCommand {
	ips := scaner.Scan()
	ipArr := make([]nodeCommand, len(ips))
	for i, ip := range ips {
		ipArr[i] = nodeCommand{
			ip: ip,
		}
	}
	return ipArr
}

func renderList(ipArray []nodeCommand) {
	for _, v := range ipArray {
		fmt.Println(v.ip)
	}
}

func wait() {
	listener, err := tcp.Listen()
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
		if savedPeer = peers.CheckIncomingPeer(ipAddr); savedPeer != nil {
			incomingPeer = savedPeer.Name
		} else {
			incomingPeer = ipAddr
		}
		fmt.Printf("request from %s\n", incomingPeer)
		fmt.Println(savedPeer)

		// respond with handshake if want to connect
		// save addr locally and named it
		// start exchange
	}
}

func connectToPeer(ip string) {
	conn, _ := tcp.Connect(ip)
	fmt.Println(conn)
}

func PromtSaveRequest(ip string) *peers.SavedPeer {
	var peer *peers.SavedPeer
	peers.RegisterPeer(ip, "test peer")
	return peer
}
