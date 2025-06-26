package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"slices"
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
	addrChan  chan string
	close     chan struct{}
	addresses []string
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

	monitor(icmpConn, ip, bytes)

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
	defer icmpListen.Close()

	var bytes []byte
	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	syncPipleChan := newSyncPipeChan()
	go syncPipleChan.processAddresses()
	go syncPipleChan.read(icmpListen)
	for i := ipStartByte; i <= ipEndByte; i++ {
		ip := "192.168.1." + strconv.Itoa(i)
		go monitor(icmpListen, ip, bytes)
	}
	syncPipleChan.sync.Wait()
	fmt.Println("closing")
	syncPipleChan.close <- struct{}{}
	log.Println("result addresses ", syncPipleChan.addresses)
}

func monitor(conn *icmp.PacketConn, ip string, msg []byte) *net.IPAddr {
	raddr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		log.Printf("error resolving raddr %e\n", err)
		return nil
	}

	for i := 0; i < 3; i++ {

		conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 200))
		_, err := conn.WriteTo(msg, raddr)
		if err != nil {
			log.Printf("error writing to %s", raddr.IP.String())
			continue
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
		addrChan: make(chan string),
		close:    make(chan struct{}),
	}
}

func (pipe *syncPipeChannel) processAddresses() {
	for {
		select {
		case addr := <-pipe.addrChan:
			fmt.Println("match")
			if !slices.Contains(pipe.addresses, addr) {
				pipe.addresses = append(pipe.addresses, addr)
			}
		case <-pipe.close:
			close(pipe.addrChan)
			return
		}
	}
}

func newDuration() time.Duration {
	return time.Until(time.Now().Add(1000 * time.Millisecond))
}

func (pipe *syncPipeChannel) read(conn *icmp.PacketConn) {
	var buffer []byte = make([]byte, 1024)
	timeout := time.NewTimer(newDuration())
	var timeoutOccured bool
	go func() {
		for {
			select {
			case <-timeout.C:
				timeoutOccured = true
			}
		}
	}()
	for {
		if timeoutOccured {
			fmt.Println("TIMEOUT")
			break
		}
		fmt.Println("start")
		pipe.sync.Add(1)

		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 300))
		if err != nil {
			log.Println(err)
		}
		n, peer, err := conn.ReadFrom(buffer)
		fmt.Println("wait")
		if err != nil {
			if peer != nil {
				// log.Printf("error reading from %s\n", peer.String())
				continue
			}
		}

		reply, err := icmp.ParseMessage(1, buffer[:n])
		if err != nil {
			// log.Printf("error parsing icmp reply %e\n", err)
		}

		if reply != nil {
			switch reply.Type {
			case ipv4.ICMPTypeEchoReply:
				fmt.Printf("Repliled to icmp from %s\n", peer.String())
				if peer != nil {
					timeout.Reset(newDuration())
					pipe.addrChan <- peer.String()
				}
			case ipv4.ICMPTypeParameterProblem:
				// fmt.Printf("ICMPTypeParameterProblem %s\n", peer.String())
			case ipv4.ICMPTypeDestinationUnreachable:
				// fmt.Printf("Destination is unreachable %s\n", peer.String())

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

// make cli
// on available addr list make possible to choose between them and request connection on port :??
// after ? connection established
// make possible to choose file
// send it
