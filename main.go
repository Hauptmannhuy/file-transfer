package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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

type syncPipeChannel struct {
	sync      *sync.WaitGroup
	addrChan  chan *net.IPAddr
	close     chan struct{}
	addresses []*net.IPAddr
}

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
	if len(os.Args) >= 2 {
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

	var bytes []byte
	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	monitor(icmpConn, ip, bytes, nil)

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

	icmpListen, err := icmp.ListenPacket("ip4:icmp", localAddr.IP.String())
	if err != nil {
		log.Fatalf("error establishing icmp %e", err)
	}

	var bytes []byte
	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	syncPipleChan := newSyncPipeChan()
	go syncPipleChan.processAddresses()
	for i := ipStartByte; i <= ipEndByte; i++ {
		syncPipleChan.sync.Add(1)
		ip := "192.168.1." + strconv.Itoa(i)
		go monitor(icmpListen, ip, bytes, syncPipleChan)
		syncPipleChan.sync.Wait()
	}
	fmt.Println("closing")
	syncPipleChan.close <- struct{}{}
	defer icmpListen.Close()
	log.Println("result addresses ", syncPipleChan.addresses)
}

func monitor(conn *icmp.PacketConn, ip string, msg []byte, syncPipe *syncPipeChannel) *net.IPAddr {
	raddr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		log.Printf("error resolving raddr %e\n", err)
		return nil
	}
	if syncPipe != nil {
		defer syncPipe.sync.Done()
	}

	var buffer []byte = make([]byte, 1024)
	for i := 0; i < 3; i++ {

		_, err := conn.WriteTo(msg, raddr)
		if err != nil {
			log.Printf("error writing to %s", raddr.IP.String())
			return nil
		}

		err = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 50))
		if err != nil {
			log.Println(err)
			return nil
		}

		n, peer, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Printf("error reading from %s\n", raddr.IP.String())
			return nil
		}

		reply, err := icmp.ParseMessage(1, buffer[:n])
		if err != nil {
			log.Printf("error parsing icmp reply %e\n", err)
			return nil
		}

		if reply != nil {

			switch reply.Type {
			case ipv4.ICMPTypeEchoReply:
				fmt.Printf("Repliled to icmp from %s\n", peer.String())
				if syncPipe != nil {
					syncPipe.addrChan <- raddr
				}
				return raddr
			case ipv4.ICMPTypeParameterProblem:
				fmt.Printf("ICMPTypeParameterProblem %s\n", peer.String())
			case ipv4.ICMPTypeDestinationUnreachable:
				fmt.Printf("Destination is unreachable %s\n", peer.String())
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
	return nil
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

func newSyncPipeChan() *syncPipeChannel {
	return &syncPipeChannel{
		sync:     &sync.WaitGroup{},
		addrChan: make(chan *net.IPAddr),
		close:    make(chan struct{}),
	}
}

func (pipe *syncPipeChannel) processAddresses() {
	for {
		select {
		case <-pipe.addrChan:
			fmt.Println("match")
			pipe.addresses = append(pipe.addresses, <-pipe.addrChan)
		case <-pipe.close:
			close(pipe.addrChan)
			return
		}
	}
}
