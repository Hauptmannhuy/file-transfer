package scaner

import (
	"errors"
	"file-transfer/logger"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mdlayher/arp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var HandshakedIPs []string

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

var localAddr *net.IPNet = GetLocalHostAddr()

var icmpMsg *icmp.Message = &icmp.Message{
	Type: ipv4.ICMPTypeEcho,
	Code: 0,
	Body: &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("Hello, are you there!"),
	},
}

func arpScanLocalNetwork() []string {
	return arpScan(getNetInterface())
}

func getNetInterface() *net.Interface {
	i, err := net.Interfaces()
	if err != nil {
		log.Fatal("error getting net interfaces")
	}

	virtualIfaces := []string{
		"docker",
	}
	for _, ninterface := range i {
		flags := ninterface.Flags.String()

		if strings.Contains(flags, net.FlagLoopback.String()) {
			continue
		}

		if !strings.Contains(flags, net.FlagRunning.String()) {
			continue
		}

		containsVirtual := false
		for _, v := range virtualIfaces {
			if strings.HasPrefix(ninterface.Name, v) {
				containsVirtual = true
				break
			}
		}

		if containsVirtual {
			continue
		}

		return &ninterface
	}

	return nil
}

func arpScan(enInterface *net.Interface) []string {
	invokedAddrs := map[string]struct{}{}
	client, err := arp.Dial(enInterface)
	if err != nil {
		log.Fatal(err)
	}
	wait := sync.WaitGroup{}
	wait.Add(1)
	logger.Log.Info("start arp")
	for i := 0; i < 255; i++ {
		addr := netip.AddrFrom4([4]byte{192, 168, 1, byte(i)})
		err := client.Request(addr)

		if err != nil {
			logger.Log.Error(err.Error())
		}

	}
	logger.Log.Info("end arp")
	logger.Log.Info("wait...")

	for {
		pack, _, err := client.Read()

		if err != nil {
			var netError net.Error
			logger.Log.Info(err.Error())
			if errors.As(err, &netError) {
				if netError.Timeout() {
					logger.Log.Info("break loop")
					break
				}
			}
		}
		logger.Log.Info("arp response")
		client.SetReadDeadline(time.Now().Add(time.Millisecond * 3000))
		invokedAddrs[pack.TargetIP.String()] = struct{}{}
		logger.Log.Info("received arp address", "addr", pack.TargetIP.String())
	}

	ipAddrs := make([]string, 0, len(invokedAddrs))
	for k := range invokedAddrs {
		ipAddrs = append(ipAddrs, k)
	}
	logger.Log.Info("returning arp addresses", "result", ipAddrs)
	return ipAddrs
}

// returns list of ip separated by comma
func Scan() string {
	res := arpScanLocalNetwork()
	if len(res) == 0 {
		return localAddr.IP.String()
	}
	return strings.Join(res, ",")
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

func icmpPingLocalNetwork() string {
	var bytes []byte
	syncPipeChan := newSyncPipeChan()

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
	for i := 0; i <= 1; i++ {
		for j := 0; j <= 254; j++ {
			ip := fmt.Sprintf("192.168.%d.%d", i, j)
			go write(icmpListen, ip, bytes)
		}
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

func GetLocalHostAddr() *net.IPNet {
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
	return time.Until(time.Now().Add(1500 * time.Millisecond))
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

		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
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
