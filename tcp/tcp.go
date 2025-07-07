package tcp

import (
	"net"
	"strconv"
)

const defaultPort int = 6070

func Connect(ipStr string) (*net.TCPConn, error) {
	var ip net.IP = net.ParseIP(ipStr)
	var raddr *net.TCPAddr
	var err error

	portStr := strconv.Itoa(defaultPort)

	raddr, err = net.ResolveTCPAddr("tcp", ip.String()+":"+portStr)
	if err != nil {
		return nil, err
	}
	return net.DialTCP("tcp", nil, raddr)
}
