package ipc

import (
	scaner "file-transfer/scan"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

type IPCstate struct {
	AddressSpaceSize int
	MemoryBlock      []byte
	CmdHandler       CmdHandler
}

type CmdHandler struct {
	Queue  chan uint8
	Buffer []byte
}

const filename string = "/mySharedMem"
const memoryBlockSize = 1024 * 256

type cmdAddr uint8

const (
	CMD_TYPE_MESSAGE_ADRESS        cmdAddr = 0
	CMD_RW_STATUS_ADRESS           cmdAddr = 1
	CMD_MESSAGE_VALUE_ADRESS_START cmdAddr = 2
	CMD_MESSAGE_VALUE_ADDRESS_END  cmdAddr = 66
)

const (
	statusRW   int = 2
	statusIdle int = 0
)

type ClientCommand struct {
	cmdEnum clientCmdEnum
	fn      func(*IPCstate, ...func() error) error
}

type clientCmdEnum uint8

const (
	CMD_REQUEST_ADDRESSES clientCmdEnum = iota + 1
)

var ClientCommands []clientCmdEnum = []clientCmdEnum{
	CMD_REQUEST_ADDRESSES,
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
			Queue:  make(chan uint8),
			Buffer: block,
		},
	}, nil
}

func decode(cmd uint8) clientCmdEnum {
	switch clientCmdEnum(cmd) {
	case CMD_REQUEST_ADDRESSES:
		return CMD_REQUEST_ADDRESSES
	default:
		panic("unknown cmd")
	}
}

func (ipcState *IPCstate) ProccessQueue() {
	handler := ipcState.CmdHandler
	mutex := &sync.Mutex{}
	for {
		cmd := <-handler.Queue
		translatedCmd := decode(cmd)
		mutex.Lock()
		ipcState.MemoryBlock[CMD_RW_STATUS_ADRESS] = byte(statusRW)
		switch translatedCmd {
		case CMD_REQUEST_ADDRESSES:
			requestLocalAddresses(ipcState.CmdHandler)
		}
		ipcState.MemoryBlock[CMD_TYPE_MESSAGE_ADRESS] = 0
		ipcState.MemoryBlock[CMD_RW_STATUS_ADRESS] = byte(statusIdle)
		mutex.Unlock()
	}
}

func (ipcState *IPCstate) Listen() {
	for {
		cmd := ipcState.MemoryBlock[CMD_TYPE_MESSAGE_ADRESS]
		if cmd > 0 {
			ipcState.CmdHandler.Queue <- cmd
		}
	}
}

func requestLocalAddresses(handler CmdHandler) {
	addrs := scaner.Scan()
	j := int(CMD_MESSAGE_VALUE_ADRESS_START)
	// k := int(CMD_MESSAGE_VALUE_ADRESS_START)
	fmt.Println("addrs to send", addrs)
	for i := 0; i < len(addrs); i++ {
		message := addrs[i]
		messageLen := len(message)

		handler.Buffer[j] = byte(messageLen)
		copy(handler.Buffer[j+1:messageLen+j+1], []byte(message))
		fmt.Println("copied ", string(handler.Buffer[j+1:messageLen+j+1]))
		fmt.Println(handler.Buffer[j+1 : messageLen+j+1])
		j = j + messageLen + 1
	}
}
