package main

import (
	"file-transfer/ipc"
	"file-transfer/server"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	exitSignal := make(chan os.Signal, 1)
	ipcState, err := ipc.InitIPC()
	if err != nil {
		panic(err)
	}
	fmt.Println("HEEEY")
	go func() {
		for {
			fmt.Println(ipcState.MemoryBlock)
			time.Sleep(time.Second * 10)
		}
	}()
	server.Start(ipcState)
	signal.Notify(exitSignal, syscall.SIGTERM)
	<-exitSignal
}
