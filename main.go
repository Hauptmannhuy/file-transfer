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
	server.Start(ipcState)
	signal.Notify(exitSignal, syscall.SIGTERM)
	<-exitSignal
	fmt.Println("exiting")
}
