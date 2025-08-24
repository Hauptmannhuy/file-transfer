package main

import (
	"file-transfer/ipc"
	"file-transfer/server"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	exitSignal := make(chan os.Signal, 1)
	ipcState, err := ipc.InitIPC()
	if err != nil {
		panic(err)
	}
	go ipcState.Listen()
	go ipcState.ProccessQueue()
	// go func() {
	// 	for {
	// 		fmt.Println(ipcState.MemoryBlock[:66])
	// 		time.Sleep(time.Second * 10)
	// 	}
	// }()
	server.Start(ipcState)
	signal.Notify(exitSignal, syscall.SIGTERM)
	<-exitSignal
	fmt.Println("exiting")
}
