package scaner

import (
	"file-transfer/logger"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
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

func getSubnetIP(enInterface *net.Interface) (*net.IPNet, error) {
	addrs, err := enInterface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil && ipnet.IP.IsPrivate() {
			return ipnet, nil
		}
	}
	return nil, nil
}

func arpScan(enInterface *net.Interface) []string {
	invokedAddrs := map[string]struct{}{}
	client, err := arp.Dial(enInterface)
	if err != nil {
		logger.Log.Error(err.Error())
	}

	subnetIp, err := getSubnetIP(enInterface)
	var addrBuffer [4]byte
	splitted := strings.Split(subnetIp.IP.String(), ".")
	for i := range 3 {
		nIP, _ := strconv.Atoi(splitted[i])
		if err != nil {
			logger.Log.Error(err.Error())
		}
		addrBuffer[i] = byte(nIP)
	}

	if err != nil {
		logger.Log.Error(err.Error())
	}

	wait := sync.WaitGroup{}
	wait.Add(1)
	logger.Log.Info("start arp")

	for i := range 255 {
		addrBuffer[3] = byte(i)
		addr := netip.AddrFrom4(addrBuffer)
		err := client.Request(addr)

		if err != nil {
			logger.Log.Error(err.Error())
		}

	}
	logger.Log.Info("end arp")
	logger.Log.Info("wait...")

	timer := time.NewTimer(time.Second * 3)

	var timeout bool
	go func() {
		<-timer.C
		timeout = true
	}()

	for {
		pack, _, err := client.Read()

		logger.Log.Info("arp response")

		if err != nil {
			logger.Log.Error(err.Error())
		}

		invokedAddrs[pack.TargetIP.String()] = struct{}{}
		logger.Log.Info("received arp address", "addr", pack.TargetIP.String())
		timer.Reset(time.Second * 3)

		if timeout {
			logger.Log.Info("break loop")
			break
		}

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

		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil && ipnet.IP.IsPrivate() {
			localAddr = ipnet
			break
		}
	}
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
