package main

import (
	"file-transfer/bridge"
	"file-transfer/ipc"
	"file-transfer/logger"
	server "file-transfer/peer"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.InitLogger()
	exitSignal := make(chan os.Signal, 1)
	readWriteMemoryBridgeEndpoint := make(chan *bridge.BridgeMessage)
	readWriteNetworkBridgeEndpoint := make(chan *bridge.BridgeMessage)
	p2pServer := server.InitServer(bridge.InitBridge(readWriteNetworkBridgeEndpoint, readWriteMemoryBridgeEndpoint))

	ipcBridge := bridge.InitBridge(readWriteMemoryBridgeEndpoint, readWriteNetworkBridgeEndpoint)
	serverEventFd, err := ipc.InitEventFd()
	if err != nil {
		panic(err)
	}

	uiEventFd, err := ipc.InitEventFd()
	if err != nil {
		panic(err)
	}

	fmt.Println(serverEventFd, uiEventFd)

	config := ipc.InitConfigIPC{
		ServerEventFd: serverEventFd,
		GuiEventFd:    uiEventFd,
		Bridge:        ipcBridge,
	}

	defer func() {
		if p := recover(); p != nil {
			panicErr, ok := p.(error)
			if ok {
				logger.Error("error start application", panicErr)
			}
		}
		err := ipc.UnlinkShmMem()
		if err != nil {
			logger.Log.Error(err.Error())
		}

		err = ipc.CloseEventFds(config)
		if err != nil {
			logger.Log.Error(err.Error())
		}

		logger.Log.Info("shutdown")
	}()

	ipcState, err := ipc.InitIPC(config)
	if err != nil {
		panic(err)
	}

	signal.Notify(exitSignal, syscall.SIGTERM)
	signal.Notify(exitSignal, syscall.SIGINT)

	go ipcState.Listen()
	go ipcState.ProccessQueue()
	go p2pServer.Listen()
	go ipcState.WatchGUIhealth(exitSignal)

	logger.Log.Info("server is working")
	<-exitSignal
}
