package peers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

const ipStorageFname string = "ips"

type SavedPeer struct {
	Name string
	IP   string
}

func RegisterPeer(ip, name string) *SavedPeer {
	f, err := os.Open(ipStorageFname)
	if err != nil {
		log.Fatal(err)
	}

	wr := bufio.NewWriter(f)
	n, err := wr.Write([]byte(fmt.Sprintf("%s:%s", ip, name)))
	if err != nil {
		log.Println(err)
		return nil
	}
	fmt.Println(n)
	return &SavedPeer{
		Name: name,
		IP:   ip,
	}
}

func CheckIncomingPeer(ip string) *SavedPeer {
	var line []byte
	var err error
	f, err := os.Open(ipStorageFname)
	if err != nil {
		log.Fatal(err)
	}

	r := bufio.NewReader(f)

	for {
		line, err = r.ReadSlice(byte('\n'))
		if err.Error() == "EOF" {
			return nil
		}
		if len(line) != 0 {
			strLine := string(line)
			value := strings.Split(strLine, ":")
			if len(strLine) != 2 {
				panic("fetched saved peer should be 2 length long")
			}
			if value[0] == ip {
				return &SavedPeer{
					IP:   ip,
					Name: value[1],
				}
			}
		}
	}
}
