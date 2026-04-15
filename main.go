package main

import (
	"file-transfer/core"
	"file-transfer/events"
	"file-transfer/ipc"
	"file-transfer/logger"
	server "file-transfer/network"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.InitLogger()
	exitSignal := make(chan os.Signal, 1)
	ipcIn := make(chan *events.EventMsg)
	ipcOut := make(chan *events.EventMsg)
	networkIn := make(chan *events.EventMsg)
	networkOut := make(chan *events.EventMsg)
	p2pServer := server.InitServer(networkIn, networkOut)
	dispatcher := core.InitializeDispatcher(ipcIn,
		ipcOut,
		networkIn,
		networkOut)

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
		Ingoing:       ipcIn,
		Outgoing:      ipcOut,
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

	go dispatcher.Dispatch()
	go ipcState.Listen()
	go ipcState.ProccessQueue()
	go p2pServer.Listen()
	go p2pServer.ProccessQueue()
	go ipcState.WatchGUIhealth(exitSignal)

	logger.Log.Info("server is working")
	<-exitSignal
}
