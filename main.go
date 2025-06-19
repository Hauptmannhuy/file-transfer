package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	ipAdressStart = "192.168.1.100"
	ipAdressEnd   = "192.168.1.200"
)

func main() {
	ipStartByte, err := strconv.Atoi(getLastIpByte(ipAdressStart))
	if err != nil {
		panic(err)
	}
	ipEndByte, err := strconv.Atoi(getLastIpByte(ipAdressEnd))
	if err != nil {
		panic(err)
	}

	for i := ipStartByte; i <= ipEndByte; i++ {
		raddr := net.ParseIP("192.168.1." + strconv.Itoa(i))
		conn, err := net.DialIP("ip4:1", &net.IPAddr{IP: getLocalHostAddress().IP}, &net.IPAddr{IP: raddr})
		if err != nil {
			log.Fatal(err)
		}

		go monitor(conn)
	}

}

func getLastIpByte(ip string) string {
	after, _ := strings.CutPrefix(ip, "192.168.1.")
	return after
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
	fmt.Println(localAddr)
	return localAddr
}

func monitor(conn *net.IPConn) {
	conn.Write([]byte("hello"))
	time.Sleep(time.Second * 5)
}
