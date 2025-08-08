package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type IPCstate struct {
	AddressSpaceSize int
	MemoryBlock      []byte
	CmdHandler       CmdHandler
}

type CmdHandler struct {
	Queue chan uint8
}

type RawClientCommandMessage struct {
	Adress uint
	Status uint8
}

type ClientCommand uint8

const (
	RequestLocalAdressesCmd ClientCommand = iota
)

const filename string = "/mySharedMem"
const memoryBlockSize = 4096

const (
	CMD_REQUEST_ADDRESSES_LOC = 20
)

var CommandAddresesLocations []int = []int{
	CMD_REQUEST_ADDRESSES_LOC,
}

func InitIPC() (*IPCstate, error) {
	f, err := os.OpenFile(filepath.Join("/dev/shm/", filename), os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("err opening file: %v", err)
	}

	block, err := syscall.Mmap(int(f.Fd()), 0, memoryBlockSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("err mapping addr space: %v", err)
	}

	return &IPCstate{
		AddressSpaceSize: memoryBlockSize,
		MemoryBlock:      block,
		CmdHandler: CmdHandler{
			Queue: make(chan uint8),
		},
	}, nil
}

func CheckRequestCommands(ipcState *IPCstate) {
	for _, addr := range CommandAddresesLocations {
		val := ipcState.MemoryBlock[addr]
		if val != 0 {
			fmt.Println("received client cmd")
		}
	}
}

func ProccessQueue(handler CmdHandler) {
	for {
		cmd := <-handler.Queue

	}
}
