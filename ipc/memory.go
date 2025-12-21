package ipc

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"file-transfer/bridge"
	"file-transfer/logger"
	scaner "file-transfer/scan"
)

type IPCstate struct {
	AddressSpaceSize int
	MemoryBlock      []byte
	cmdHandler       cmdHandler
	networkBridge    bridge.Bridge
	frontBlock       controlBlock
	backBlock        controlBlock
	uiEventFd        eventfd
	serverEventFd    eventfd
	ShmFile          *os.File
}

type controlBlock struct {
	memory      []byte
	writeOffset uint32
	readOffset  uint32
	rdwrStatus  uint8
}

type cmdHandler struct {
	Queue  chan cmdMessage
	Buffer []byte
}

const (
	filename        string = "mySharedMem"
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
	cmdEnum bridge.ClientCmdEnum
	fn      func(*IPCstate, ...func() error) error
}

type cmdMessage struct {
	cmdType        uint32
	cmdPayloadSize uint32
	payload        []byte
}

func newMessage(cmdType uint32, payload []byte) *cmdMessage {
	return &cmdMessage{
		cmdType:        cmdType,
		payload:        payload,
		cmdPayloadSize: uint32(len(payload)),
	}
}

var ClientCommands []bridge.ClientCmdEnum = []bridge.ClientCmdEnum{
	bridge.CmdRequestAddresses,
}

func InitIPC(bridge bridge.Bridge) (*IPCstate, error) {
	f, err := os.OpenFile(filepath.Join("/dev/shm/", filename), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("error creating fd for shared memory segment, %s", err.Error())
	}

	if err = f.Truncate(memoryBlockSize); err != nil {
		return nil, nil
	}

	if err = syscall.Ftruncate(int(f.Fd()), memoryBlockSize); err != nil {
		return nil, err
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

	// map write and read offset indexes
	offsetStart := 10
	ptr1 := (*uint32)(unsafe.Pointer(&fblockControl.memory[indexWriteOffset]))
	ptr2 := (*uint32)(unsafe.Pointer(&fblockControl.memory[indexReadOffset]))
	ptr3 := (*uint32)(unsafe.Pointer(&bblockControl.memory[indexWriteOffset]))
	ptr4 := (*uint32)(unsafe.Pointer(&bblockControl.memory[indexReadOffset]))

	ptrArr := [4]*uint32{ptr1, ptr2, ptr3, ptr4}

	for _, ptr := range ptrArr {
		*ptr = uint32(offsetStart)
	}

	serverEventFd, err := initEventFd()
	if err != nil {
		return nil, err
	}

	uiEventFd, err := initEventFd()
	if err != nil {
		return nil, err
	}
	fmt.Println(serverEventFd, uiEventFd)

	serverFd := os.NewFile(uintptr(serverEventFd), "server_fd")
	uiFd := os.NewFile(uintptr(uiEventFd), "ui_fd")

	cmd := exec.Command("./gui/out", "3", "4")
	cmd.ExtraFiles = []*os.File{serverFd, uiFd}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	ipc := &IPCstate{
		AddressSpaceSize: memoryBlockSize,
		MemoryBlock:      block,
		cmdHandler: cmdHandler{
			Queue:  make(chan cmdMessage),
			Buffer: block,
		},
		networkBridge: bridge,
		frontBlock:    fblockControl,
		backBlock:     bblockControl,
		ShmFile:       f,
		serverEventFd: serverEventFd,
		uiEventFd:     uiEventFd,
	}

	ipc.identifyHost(scaner.GetLocalHostAddr())
	return ipc, nil
}

func UnlinkShmMem() error {
	err := os.Remove(filepath.Join("/dev/shm/", filename))
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}
	logger.Log.Info("unlinking shared memory")
	return nil
}

func (ipcState *IPCstate) ProccessQueue() {
	ipcHandler := ipcState.cmdHandler
	mutex := &sync.Mutex{}
	for {
		select {
		case ipcCmd := <-ipcHandler.Queue:
			processIpcCmd(ipcState, mutex, ipcCmd)
		case <-ipcState.networkBridge.Get():
			log.Println("received message from net")
		}

	}
}

func processIpcCmd(ipc *IPCstate, mutex *sync.Mutex, ipcCmd cmdMessage) {
	mutex.Lock()
	var data any

	clientCmdType := bridge.ClientCmdEnum(ipcCmd.cmdType)

	switch clientCmdType {
	case bridge.CmdRequestAddresses:
		data = scaner.Scan()
	case bridge.CmdRequestP2P:
		log.Println("sending message to network")
		ip := string(ipcCmd.payload)
		ipc.networkBridge.Send(&bridge.BridgeMessage{
			RequestP2P: &ip,
			Type:       clientCmdType,
		})

	case bridge.CmdSendFile:
		// filePath := string(ipcCmd.payload)
		// get file data and name...
		msg := &bridge.BridgeMessage{
			Type: clientCmdType,
			SendFileData: &bridge.SendFileData{
				Data:     []byte{},
				FileName: "name",
			},
		}
		ipc.networkBridge.Send(msg)
	default:
		log.Printf("unknown command: %d \n", ipcCmd.cmdType)
	}
	var buffer []byte
	if data != nil {
		buffer = encodePayload(data)
	}
	responseMsg := newMessage(ipcCmd.cmdType, buffer)
	ipc.sendMessage(responseMsg)
	ipc.MemoryBlock[CMD_RW_STATUS_ADRESS] = byte(statusIdle)
	mutex.Unlock()
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
		logger.Log.Info("decoded message type %d", msg.cmdType)
		updateSize := getUpdateSize(msg)
		logger.Log.Info("%d", updateSize)
		UpdateWriteOffset(ipcState.backBlock.memory, updateSize)
		ClearQueue(ipcState.backBlock.memory, offset, offset+updateSize)
		ipcState.cmdHandler.Queue <- *msg
	}
}

func DecodeCommandMsg(memory []byte, offset uint32) (*cmdMessage, error) {

	commandType := ReadFourBytes(memory, offset)
	payloadSize := ReadFourBytes(memory, offset+sizeOfUint32)
	var messagePayload []byte = make([]byte, payloadSize)
	payloadOffsetStart := offset + (sizeOfUint32 * 2)
	payloadOffsetEnd := payloadOffsetStart + payloadSize
	copy(messagePayload, memory[payloadOffsetStart:payloadOffsetEnd])

	if (commandType | payloadSize) == 0 {
		return nil, nil
	}
	if commandType == 0 && payloadSize > 0 {
		return nil, fmt.Errorf("invalid message: size=%d but type=0", payloadSize)
	}
	newMessage(commandType, nil)
	return &cmdMessage{
		cmdType:        commandType,
		cmdPayloadSize: payloadSize,
		payload:        messagePayload,
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
	logger.Log.Info("clear queue with offset start-end:", offsetStart, offsetEnd)
	logger.Log.Info("memory size", len(memory))
	for i := offsetStart; i < offsetEnd; i++ {
		memory[i] = 0
	}
	logger.Log.Info("queue cleared")
}

func (ipcState *IPCstate) sendMessage(message *cmdMessage) {
	offset := GetWriteOffset(ipcState.frontBlock.memory)
	handler := ipcState.cmdHandler
	j := int(offset)
	binary.NativeEndian.PutUint32(handler.Buffer[j:], message.cmdType)
	binary.NativeEndian.PutUint32(handler.Buffer[j+4:], message.cmdPayloadSize)
	copy(handler.Buffer[j+8:], message.payload)
}

func encodePayload(data any) []byte {
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

func getUpdateSize(msg *cmdMessage) uint32 {
	return msg.cmdPayloadSize + uint32(unsafe.Sizeof(msg.cmdType))
}

func (ipcState *IPCstate) identifyHost(localHostAddr *net.IPNet) {
	addr := localHostAddr.IP.String()
	buffer := encodePayload(addr)
	message := newMessage(uint32(bridge.CmdIdentifyHost), buffer)
	ipcState.sendMessage(message)
	UpdateWriteOffset(ipcState.backBlock.memory, getUpdateSize(message))
}
