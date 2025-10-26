package scaner

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

var HandshakedIPs []string

const (
	ipAdressStart = "192.168.1.100"
	ipAdressEnd   = "192.168.1.200"
)

const (
	defaultIcmpConf = "ip:icmp"
)

type syncPipeChannel struct {
	sync      *sync.WaitGroup
	mutex     *sync.Mutex
	addrChan  chan string
	close     chan struct{}
	addresses []string
}

var localAddr *net.IPNet = GetLocalHostAddress()

var icmpMsg *icmp.Message = &icmp.Message{
	Type: ipv4.ICMPTypeEcho,
	Code: 0,
	Body: &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("Hello, are you there!"),
	},
}

// returns list of ip separated by comma
func Scan() string {
	return pingLocalNetwork()
}

func getLastIpByte(ip string) string {
	after, _ := strings.CutPrefix(ip, "192.168.1.")
	return after
}

func ping(ip string) {
	var bytes []byte
	icmpConn, err := icmp.ListenPacket(defaultIcmpConf, localAddr.IP.String())
	if err != nil {
		panic(err)
	}

	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	write(icmpConn, ip, bytes)

}

func pingLocalNetwork() string {
	var bytes []byte
	syncPipeChan := newSyncPipeChan()

	ipStartByte, err := strconv.Atoi(getLastIpByte(ipAdressStart))
	if err != nil {
		panic(err)
	}

	ipEndByte, err := strconv.Atoi(getLastIpByte(ipAdressEnd))
	if err != nil {
		panic(err)
	}

	icmpListen, err := icmp.ListenPacket(defaultIcmpConf, localAddr.IP.String())
	if err != nil {
		log.Fatalf("error establishing icmp %e", err)
	}
	defer icmpListen.Close()

	bytes, err = icmpMsg.Marshal(bytes)
	if err != nil {
		panic(err)
	}

	syncPipeChan.sync.Add(1)
	go syncPipeChan.processAddresses()
	go syncPipeChan.read(icmpListen)
	for i := ipStartByte; i <= ipEndByte; i++ {
		ip := "192.168.1." + strconv.Itoa(i)
		go write(icmpListen, ip, bytes)
	}
	syncPipeChan.sync.Wait()
	err = icmpListen.Close()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("closing")
	syncPipeChan.close <- struct{}{}
	log.Println("result addresses ", syncPipeChan.addresses)
	return strings.Join(syncPipeChan.addresses, ",")
}

func write(conn *icmp.PacketConn, ip string, msg []byte) *net.IPAddr {
	raddr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		log.Printf("error resolving raddr %e\n", err)
		return nil
	}

	for i := 0; i < 3; i++ {
		conn.SetWriteDeadline(time.Now().Add(time.Millisecond * 200))
		_, err := conn.WriteTo(msg, raddr)
		if err != nil {
			// log.Printf("error writing to %s", raddr.IP.String())
			continue
		}

	}

	return nil
}

func GetLocalHostAddress() *net.IPNet {
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
	// fmt.Println("local address is ", localAddr.IP.String())
	return localAddr
}

func newSyncPipeChan() *syncPipeChannel {
	return &syncPipeChannel{
		sync:     &sync.WaitGroup{},
		mutex:    &sync.Mutex{},
		addrChan: make(chan string),
		close:    make(chan struct{}),
	}
}

func (pipe *syncPipeChannel) processAddresses() {
	for {
		select {
		case addr := <-pipe.addrChan:
			// fmt.Println("match")
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
	return time.Until(time.Now().Add(800 * time.Millisecond))
}

func (pipe *syncPipeChannel) read(conn *icmp.PacketConn) {
	var buffer []byte = make([]byte, 1024)
	timeout := time.NewTimer(newDuration())
	timesOccured := 3

	go func() {
		for range timeout.C {
			fmt.Println("TICK DETECTED, times left ", timesOccured)
			timesOccured -= 1
			if timesOccured == 0 {
				return
			}
			timeout.Reset(newDuration())
		}
	}()

	for {
		if timesOccured == 0 {
			fmt.Println("TIMEOUT")
			pipe.sync.Done()
			return
		}

		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 300))
		if err != nil {
			continue
		}
		n, peer, err := conn.ReadFrom(buffer)
		if err != nil {
			if peer != nil {
				continue
			}
		}

		reply, err := icmp.ParseMessage(1, buffer[:n])
		if err != nil {
			continue
		}

		if reply != nil {
			switch reply.Type {
			case ipv4.ICMPTypeEchoReply:
				if peer != nil {
					timeout.Reset(newDuration())
					pipe.addrChan <- peer.String()
					if timesOccured < 3 {
						timesOccured++
					}
				}
			}
		}
	}

}
