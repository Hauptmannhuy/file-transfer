package ipc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"log/slog"
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
	guiBuffer        controlBlock
	serverBuffer     controlBlock
	uiEventFd        eventfd
	serverEventFd    eventfd
	guiProcess       *os.Process
	ShmFile          *os.File

	guiDispatch chan *cmdMessage
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

type InitConfigIPC struct {
	ServerEventFd eventfd
	GuiEventFd    eventfd
	Bridge        bridge.Bridge
}

var ClientCommands []bridge.ClientCmdEnum = []bridge.ClientCmdEnum{
	bridge.CmdRequestAddresses,
}

func InitIPC(config InitConfigIPC) (*IPCstate, error) {
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

	serverFd := os.NewFile(uintptr(config.ServerEventFd), "server_fd")
	uiFd := os.NewFile(uintptr(config.GuiEventFd), "ui_fd")

	cmd := exec.Command("./gui/out", "3", "4")
	cmd.ExtraFiles = []*os.File{serverFd, uiFd}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			logger.Log.Error("GUI exited with an error", "error", err.Error())
		}
	}()

	ipc := &IPCstate{
		AddressSpaceSize: memoryBlockSize,
		MemoryBlock:      block,
		cmdHandler: cmdHandler{
			Queue:  make(chan cmdMessage),
			Buffer: block,
		},
		networkBridge: config.Bridge,
		guiBuffer:     fblockControl,
		serverBuffer:  bblockControl,
		ShmFile:       f,
		serverEventFd: config.ServerEventFd,
		uiEventFd:     config.GuiEventFd,
		guiProcess:    cmd.Process,
		guiDispatch:   make(chan *cmdMessage),
	}

	go ipc.dispatchToGUI()
	ipc.identifyHost(scaner.GetLocalHostAddr())
	return ipc, nil
}

func (ipc *IPCstate) dispatchToGUI() {
	for msg := range ipc.guiDispatch {
		offset := GetWriteOffset(ipc.guiBuffer.memory)
		handler := ipc.cmdHandler
		j := int(offset)
		binary.NativeEndian.PutUint32(handler.Buffer[j:], msg.cmdType)
		binary.NativeEndian.PutUint32(handler.Buffer[j+4:], msg.cmdPayloadSize)
		copy(handler.Buffer[j+8:], msg.payload)
		// GUI listens to server with server event fd to handle events, so we write to server eventfd
		_, err := ipc.serverEventFd.Write(1)
		if err != nil {
			logger.Log.Error("error signal to GUI eventfd", "error", slog.AnyValue(err))
		}
		logger.Log.Info("message sent to GUI", "cmd_type", msg.cmdType, "payload", string(msg.payload))
		UpdateWriteOffset(ipc.serverBuffer.memory, getUpdateSize(msg))
	}
}

func (ipc *IPCstate) WatchGUIhealth(exitSignal chan os.Signal) {
	state, err := ipc.guiProcess.Wait()
	if err != nil {
		logger.Log.Error("GUI exited with an errror", "error_code", state.ExitCode(), "state", state.String())
		exitSignal <- syscall.SIGTERM
	}
}

func UnlinkShmMem() error {
	err := os.Remove(filepath.Join("/dev/shm/", filename))
	if err != nil {
		return err
	}
	logger.Log.Info("unlinking shared memory")
	return nil
}

func CloseEventFds(config InitConfigIPC) error {
	var errs []error

	if err := syscall.Close(int(config.GuiEventFd)); err != nil {
		errs = append(errs, err)
	}

	if err := syscall.Close(int(config.ServerEventFd)); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (ipc *IPCstate) ProccessQueue() {
	ipcHandler := ipc.cmdHandler
	mutex := &sync.Mutex{}
	for {
		select {
		case ipcCmd := <-ipcHandler.Queue:
			ipc.processIpcCmd(mutex, ipcCmd)
		case netBridgeMsg := <-ipc.networkBridge.Get():
			log.Println("received message from net")
			ipc.processNetCmd(netBridgeMsg)
		}

	}
}

func (ipc *IPCstate) processNetCmd(netBridgeMsg *bridge.BridgeMessage) {
	switch netBridgeMsg.Type {
	case bridge.AcceptP2P:
		ipc.guiDispatch <- newMessage(uint32(netBridgeMsg.Type), []byte(*netBridgeMsg.RequestP2P))
	}
}

func (ipc *IPCstate) processIpcCmd(mutex *sync.Mutex, ipcCmd cmdMessage) {
	mutex.Lock()
	var data any

	clientCmdType := bridge.ClientCmdEnum(ipcCmd.cmdType)

	switch clientCmdType {
	case bridge.CmdRequestAddresses:
		data = scaner.Scan()
	case bridge.CmdRequestP2P:
		ip := string(ipcCmd.payload)
		logger.Log.Info("bridge.CmdRequestP2P", "ip", ip)
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
		responseMsg := newMessage(ipcCmd.cmdType, buffer)
		ipc.guiDispatch <- responseMsg
	}
	mutex.Unlock()
}

func (ipcState *IPCstate) Listen() {
	for {
		_, err := ipcState.uiEventFd.Read()
		if err != nil {
			logger.Log.Error("error signal to GUI eventfd", "error", slog.AnyValue(err))
			os.Exit(1)
		}

		logger.Log.Info("received event from GUI")

		offset := GetWriteOffset(ipcState.serverBuffer.memory)
		msg, err := decodeMessage(ipcState.serverBuffer.memory, offset)
		if err != nil {
			log.Fatal(err.Error())
		}
		if msg == nil {
			continue
		}
		logger.Log.Info("decoded message", "cmd_type", msg.cmdType)
		updateSize := getUpdateSize(msg)
		logger.Log.Info("%d", "update_", updateSize)
		UpdateWriteOffset(ipcState.serverBuffer.memory, updateSize)
		ClearQueue(ipcState.serverBuffer.memory, offset, offset+updateSize)
		ipcState.cmdHandler.Queue <- *msg
	}
}

func decodeMessage(memory []byte, offset uint32) (*cmdMessage, error) {
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

	logger.Log.Info("decoded message from GUI", "commandType", commandType, "payload size", payloadSize, "payload", string(messagePayload))

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
	logger.Log.Info("clear queue:", "offset_start", offsetStart, "offset_end", offsetEnd)
	for i := offsetStart; i < offsetEnd; i++ {
		memory[i] = 0
	}
	logger.Log.Info("queue cleared")
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
	ipcState.guiDispatch <- message

}
