package ipc

import (
	"encoding/binary"
	scaner "file-transfer/scan"
	"fmt"
	"log"
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
	Queue  chan cmdMessage
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
	CMD_RW_STATUS_ADRESS cmdAddr = 1
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
	CmdRequestAddresses clientCmdEnum = iota + 1
)

type serverCmdEnum uint8

const (
	CmdIdentifyHost serverCmdEnum = iota + 1
)

var ClientCommands []clientCmdEnum = []clientCmdEnum{
	CmdRequestAddresses,
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
			Queue:  make(chan cmdMessage),
			Buffer: block,
		},
		frontBlock: fblockControl,
		backBlock:  bblockControl,
	}, nil
}

func (ipcState *IPCstate) ProccessQueue() {
	handler := ipcState.CmdHandler
	mutex := &sync.Mutex{}
	for {
		cmd := <-handler.Queue
		mutex.Lock()
		var data any
		switch uint32(cmd.cmdType) {
		case uint32(CmdRequestAddresses):
			data = scaner.Scan()
		}

		buffer := encodeMessage(data)

		ipcState.sendMessage(cmd.cmdType, buffer)
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
			log.Fatal(err.Error())
		}
		if msg == nil {
			continue
		}
		fmt.Println("decoded message type", msg.cmdType)
		updateSize := getUpdateSize(*msg)
		UpdateWriteOffset(ipcState.backBlock.memory, updateSize)
		ClearQueue(ipcState.backBlock.memory, offset, offset+updateSize)
		ipcState.CmdHandler.Queue <- *msg
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

func (ipcState *IPCstate) sendMessage(msgType uint32, data []byte) {
	offset := GetWriteOffset(ipcState.frontBlock.memory)
	handler := ipcState.CmdHandler
	j := int(offset)
	binary.NativeEndian.PutUint32(handler.Buffer[j:], msgType)
	binary.NativeEndian.PutUint32(handler.Buffer[j+4:], uint32(len(data)))
	copy(handler.Buffer[j+8:], data)
}

func encodeMessage(data any) []byte {
	var length int

	switch d := data.(type) {
	case string:
		length = len((d))
		data = []byte(d)
	}

	buffer := make([]byte, length)
	_, err := binary.Encode(buffer, binary.NativeEndian, data)
	if err != nil {
		fmt.Printf("error encoding message %s", err.Error())
		os.Exit(-1)
	}
	return buffer
}

func getUpdateSize(msg cmdMessage) uint32 {
	return msg.cmdPayloadSize + uint32(unsafe.Sizeof(msg.cmdType))
}

func (ipcState *IPCstate) IdentifyHost(localHostAddr string) {
	buffer := encodeMessage(localHostAddr)
	ipcState.sendMessage(uint32(CmdIdentifyHost), buffer)
}
