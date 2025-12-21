package main

import (
	"file-transfer/bridge"
	"file-transfer/ipc"
	"file-transfer/logger"
	server "file-transfer/peer"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.InitLogger()
	exitSignal := make(chan os.Signal, 1)
	readWriteMemoryBridge := make(chan *bridge.BridgeMessage)
	readWriteNetworkBridge := make(chan *bridge.BridgeMessage)
	p2pServer := server.InitServer(bridge.InitBridge(readWriteNetworkBridge, readWriteMemoryBridge))

	defer func() {
		if p := recover(); p != nil {
			panicErr, ok := p.(error)
			if ok {
				logger.Error("error start application", panicErr)
			}
		}
		ipc.UnlinkShmMem()
		logger.Log.Info("shutdown")
	}()

	ipcState, err := ipc.InitIPC(bridge.InitBridge(readWriteMemoryBridge, readWriteNetworkBridge))
	if err != nil {
		panic(err)
	}

	signal.Notify(exitSignal, syscall.SIGTERM)
	signal.Notify(exitSignal, syscall.SIGINT)

	go ipcState.Listen()
	go ipcState.ProccessQueue()
	go p2pServer.Listen()

	logger.Log.Info("server is working")
	<-exitSignal
}
