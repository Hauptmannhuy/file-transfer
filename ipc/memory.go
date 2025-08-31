package ipc

import (
	scaner "file-transfer/scan"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

type IPCstate struct {
	AddressSpaceSize int
	MemoryBlock      []byte
	CmdHandler       CmdHandler
	frontBlock       controlBlock
	backBlock        controlBlock
}

type controlBlock struct {
	memory      []byte
	writeOffset uint32
	readOffset  uint32
	rdwrStatus  uint8
}

type CmdHandler struct {
	Queue  chan uint8
	Buffer []byte
}

const (
	filename        string = "/mySharedMem"
	memoryBlockSize        = 1024 * 256
	bblockAddrStart        = memoryBlockSize / 2
	fblockAddrStart        = 0
)

const sizeOfUint32 = 4

const (
	indexWriteOffset = 4
	indexReadOffset  = 0
)

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

type cmdMessage struct {
	cmdType        uint32
	cmdPayloadSize uint32
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

	fblockControl := controlBlock{
		memory: block[fblockAddrStart:bblockAddrStart],
	}

	bblockControl := controlBlock{
		memory: block[bblockAddrStart:],
	}

	return &IPCstate{
		AddressSpaceSize: memoryBlockSize,
		MemoryBlock:      block,
		CmdHandler: CmdHandler{
			Queue:  make(chan uint8),
			Buffer: block,
		},
		frontBlock: fblockControl,
		backBlock:  bblockControl,
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
		ipcState.MemoryBlock[CMD_TYPE_MESSAGE_ADRESS] = 0
		ipcState.MemoryBlock[CMD_RW_STATUS_ADRESS] = byte(statusRW)
		switch translatedCmd {
		case CMD_REQUEST_ADDRESSES:
			requestLocalAddresses(ipcState.CmdHandler)
		}
		ipcState.MemoryBlock[CMD_RW_STATUS_ADRESS] = byte(statusIdle)
		mutex.Unlock()
	}
}

// TODO: think about how to actually use read and write offsets in communication protocol
func (ipcState *IPCstate) Listen() {
	for {
		offset := GetWriteOffset(ipcState.backBlock.memory)
		msg, err := DecodeCommandMsg(ipcState.backBlock.memory, offset)
		if err != nil {
			os.Exit(1)
		}
		if msg == nil {
			continue
		}
		updateSize := msg.cmdPayloadSize + uint32(unsafe.Sizeof(msg.cmdType))
		UpdateWriteOffset(ipcState.backBlock.memory, updateSize)
		ClearQueue(ipcState.backBlock.memory, offset, offset+updateSize)
	}
}

func DecodeCommandMsg(memory []byte, offset uint32) (*cmdMessage, error) {
	commandType := ReadFourBytes(memory, offset)
	commandSize := ReadFourBytes(memory, offset+sizeOfUint32)

	if (commandType | commandSize) == 0 {
		return nil, nil
	}
	if commandType == 0 && commandSize > 0 {
		return nil, fmt.Errorf("invalid message: size=%d but type=0", commandSize)
	}
	return &cmdMessage{
		cmdType:        commandType,
		cmdPayloadSize: commandSize,
	}, nil
}

func ReadFourBytes(memory []byte, offset uint32) uint32 {
	return *(*uint32)(unsafe.Pointer(&memory[offset]))
}

func GetReadOffset(memory []byte) uint32 {
	return ReadFourBytes(memory, indexReadOffset)
}

func GetWriteOffset(memory []byte) uint32 {
	return ReadFourBytes(memory, indexWriteOffset)
}

func UpdateWriteOffset(memory []byte, size uint32) {
	p := (*uint32)(unsafe.Pointer(&memory[indexWriteOffset]))
	*p = *p + size
}

func ClearQueue(memory []byte, offsetStart, offsetEnd uint32) {
	fmt.Println(offsetStart, offsetEnd)
	for i := offsetStart; i < offsetEnd; i++ {
		memory[i] = 0
	}
}

func requestLocalAddresses(handler CmdHandler) {
	addrs := scaner.Scan()
	j := int(CMD_MESSAGE_VALUE_ADRESS_START)
	for i := 0; i < len(addrs); i++ {
		message := addrs[i]
		messageLen := len(message)
		handler.Buffer[j] = byte(messageLen)
		copy(handler.Buffer[j+1:messageLen+j+1], []byte(message))
		j = j + messageLen + 1
	}
}
