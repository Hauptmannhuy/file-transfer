package cmd

import (
	scaner "file-transfer/scan"
	"file-transfer/tcp"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
)

type nodeCommand struct {
	command *cobra.Command
	ip      string
	conn    *net.TCPConn
}

var rootCmd = &cobra.Command{
	Use:   "hugo",
	Short: "Hugo is a very fast static site generator",
	Long: `A Fast and Flexible Static Site Generator built with
                love by spf13 and friends in Go.
                Complete documentation is available at https://gohugo.io/documentation/`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	if len(scaner.HandshakedIPs) == 0 {
		fmt.Println("There is no saved hosts that you can connect to. \n Performing search...")
		connections := initList()
		renderList(connections)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initList() []nodeCommand {
	ips := scaner.Scan()
	ipArr := make([]nodeCommand, len(ips))
	for i, ip := range ips {
		conn, err := tcp.Connect(ip)
		if err != nil {
			continue
		}
		ipArr[i] = nodeCommand{
			ip:   ip,
			conn: conn,
		}
	}
	return ipArr
}

func renderList(ipArray []nodeCommand) {
	for _, v := range ipArray {
		fmt.Println(v.ip)
	}
}
