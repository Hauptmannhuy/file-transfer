package tcp

import (
	scaner "file-transfer/scan"
	"fmt"
	"net"
	"net/netip"
	"strconv"
)

const defaultPort uint16 = 6070
const connectPort uint16 = 6071

func Listen() (*net.TCPListener, error) {
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

func Connect(ipStr string) (*net.TCPConn, error) {
	var ip net.IP = net.ParseIP(ipStr)
	var raddr *net.TCPAddr
	var laddr *net.TCPAddr
	var err error

	portStr := strconv.Itoa(int(defaultPort))

	raddr, err = net.ResolveTCPAddr("tcp", ip.String()+":"+portStr)
	if err != nil {
		return nil, err
	}

	// net.TCPAddrFromAddrPort()

	laddr, err = net.ResolveTCPAddr("tcp", scaner.GetLocalHostAddress().IP.String()+":"+strconv.Itoa(int(connectPort)))
	if err != nil {
		return nil, err
	}
	fmt.Println("requesting from %s to %s", laddr.String(), raddr.String())
	return net.DialTCP("tcp", laddr, raddr)
}
