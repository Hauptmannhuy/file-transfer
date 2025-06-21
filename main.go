package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	ipAdressStart = "192.168.1.100"
	ipAdressEnd   = "192.168.1.200"
)

const (
	defaultIcmpConf = "ip4:icmp"
)

var localAddr *net.IPNet = getLocalHostAddress()

var icmpMsg *icmp.Message = &icmp.Message{
	Type: ipv4.ICMPTypeEcho,
	Code: 0,
	Body: &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("Hello, are you there!"),
	},
}

func main() {
	var ip string
	if len(os.Args) > 1 {
		ip = os.Args[1]
		ping(ip)
	} else {
		pingLocalNetwork()

	}
}

func getLastIpByte(ip string) string {
	after, _ := strings.CutPrefix(ip, "192.168.1.")
	return after
}

func ping(ip string) {
	icmpConn, err := icmp.ListenPacket(defaultIcmpConf, localAddr.IP.String())
	if err != nil {
		panic(err)
	}

	fmt.Println(ip)
	raddr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		panic(err)
	}

	var bytes []byte
	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	monitor(icmpConn, raddr, bytes)

}

func pingLocalNetwork() {
	ipStartByte, err := strconv.Atoi(getLastIpByte(ipAdressStart))
	if err != nil {
		panic(err)
	}
	ipEndByte, err := strconv.Atoi(getLastIpByte(ipAdressEnd))
	if err != nil {
		panic(err)
	}

	icmpListen, err := icmp.ListenPacket("ip4:icmp", getLocalHostAddress().IP.String())
	if err != nil {
		log.Fatalf("error establishing icmp %e", err)
	}

	var bytes []byte
	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	for i := ipStartByte; i <= ipEndByte; i++ {

		raddr, err := net.ResolveIPAddr("ip", "192.168.1."+strconv.Itoa(i))
		if err != nil {
			log.Printf("error resolving raddr %e\n", err)
		}
		go monitor(icmpListen, raddr, bytes)

	}
	defer icmpListen.Close()
}

func getLocalHostAddress() *net.IPNet {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	var localAddr *net.IPNet
	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
			localAddr = ip
			break
		}
	}
	fmt.Println("local address is ", localAddr.IP.String())
	return localAddr
}

func monitor(conn *icmp.PacketConn, raddr *net.IPAddr, msg []byte) {
	var buffer []byte = make([]byte, 1024)

	for i := 0; i < 3; i++ {

		_, err := conn.WriteTo(msg, raddr)
		if err != nil {
			log.Printf("error writing to %s", raddr.IP.String())
		}

		err = conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			log.Println(err)
		}

		n, ipAddr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Printf("error reading from %e\n", err)
		}

		reply, err := icmp.ParseMessage(1, buffer[:n])
		if err != nil {
			log.Printf("error parsing icmp reply %e\n", err)
			log.Println("reply: ", reply)
		}

		if reply != nil {

			switch reply.Type {
			case ipv4.ICMPTypeEchoReply:

				fmt.Printf("Repliled to icmp from %s\n", ipAddr.String())
			case ipv4.ICMPTypeParameterProblem:
				fmt.Printf("ICMPTypeParameterProblem %s\n", ipAddr.String())
			case ipv4.ICMPTypeDestinationUnreachable:
				fmt.Printf("Destination is unreachable %s\n", ipAddr.String())
			default:
				body, err := reply.Body.Marshal(4)
				if err != nil {
					log.Printf("errrr %e\n", err)
				}
				fmt.Println("invalid icmp reply")
				fmt.Println(reply.Type.Protocol(), reply.Code, string(body), reply.Checksum)
			}
		}
	}

}
