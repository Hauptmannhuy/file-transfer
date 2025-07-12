package cmd

import (
	"file-transfer/peers"
	scaner "file-transfer/scan"
	"file-transfer/tcp"
	"fmt"
	"log"
	"net"
)

type nodeCommand struct {
	ip   string
	conn *net.TCPConn
}

func Execute() {
	if len(scaner.HandshakedIPs) == 0 {
		fmt.Println("There is no saved hosts that you can connect to. \n Performing search...")
		connections := initList()
		renderList(connections)
	}

	go wait()

	for {
		fmt.Println("type an ip to connect")
		connectToPeer()
	}

}

func initList() []nodeCommand {
	ips := scaner.Scan()
	ipArr := make([]nodeCommand, len(ips))
	for i, ip := range ips {
		// conn, err := tcp.Connect(ip)
		// if err != nil {
		// 	continue
		// }
		ipArr[i] = nodeCommand{
			ip: ip,
			// conn: conn,
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

		ipAddr := conn.LocalAddr().String()
		if savedPeer = peers.CheckIncomingPeer(ipAddr); savedPeer != nil {
			incomingPeer = savedPeer.Name
		}
		fmt.Printf("request from %s\n", incomingPeer)
		fmt.Println(savedPeer)
		if savedPeer != nil {
			savedPeer = PromtSaveRequest(ipAddr)
			if savedPeer != nil {
				continue
			}

		}

		// respond with handshake if want to connect
		// save addr locally and named it
		// start exchange
	}
}

func connectToPeer() {
	ip := userInput()
	conn, _ := tcp.Connect(ip)
	fmt.Println(conn)
}

func PromtSaveRequest(ip string) *peers.SavedPeer {
	var peer *peers.SavedPeer
	fmt.Printf("do you want to establish connection with %s?\n type 'y' or 'n'\n", ip)
	result := userInput()
	if result == "n" {
		fmt.Println("aborting connection...")
		return nil
	} else if result == "y" {
		fmt.Println()
		name := userInput()
		peer = peers.RegisterPeer(name, ip)
	}
	return peer
}

func userInput() string {
	var ip string
	fmt.Print("input: ")
	fmt.Scan(&ip)
	return ip
}
