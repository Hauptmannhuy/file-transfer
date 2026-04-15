package ipc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"file-transfer/events"
	"file-transfer/logger"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

type IPCstate struct {
	AddressSpaceSize int
	MemoryBlock      []byte
	cmdHandler       cmdHandler
	ingoing          chan (*events.EventMsg)
	outgoing         chan (*events.EventMsg)
	guiBuffer        memoryBlock
	serverBuffer     memoryBlock
	uiEventFd        eventfd
	serverEventFd    eventfd
	guiProcess       *os.Process
	ShmFile          *os.File

	guiDispatch chan *EncodedClientMsg
}

type memoryBlock struct {
	memory      []byte
	writeOffset uint32
	readOffset  uint32
	rdwrStatus  uint8
}

type cmdHandler struct {
	Queue  chan EncodedClientMsg
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

type ClientResponse int

const (
	ClientConnectAccept = iota + 1
	ClientConnectRefuse
)

type EncodedClientMsg struct {
	CmdType        uint32
	CmdPayloadSize uint32
	Payload        []byte
}

func NewEncodedClientMsg(cmdType uint32, payload []byte) *EncodedClientMsg {
	return &EncodedClientMsg{
		CmdType:        cmdType,
		Payload:        payload,
		CmdPayloadSize: uint32(len(payload)),
	}
}

type InitConfigIPC struct {
	ServerEventFd eventfd
	GuiEventFd    eventfd
	Ingoing       chan (*events.EventMsg)
	Outgoing      chan (*events.EventMsg)
}

var ClientCommands []events.ClientCmdEnum = []events.ClientCmdEnum{
	events.CmdRequestAddresses,
}

type SerializableFields struct {
	Err            string `json:"error"`
	IP             string `json:"ip"`
	StatusResponse bool   `json:"status_response"`
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

	fblockControl := memoryBlock{
		memory: block[fblockAddrStart:bblockAddrStart],
	}

	bblockControl := memoryBlock{
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
			Queue:  make(chan EncodedClientMsg),
			Buffer: block,
		},
		ingoing:       config.Ingoing,
		outgoing:      config.Outgoing,
		guiBuffer:     fblockControl,
		serverBuffer:  bblockControl,
		ShmFile:       f,
		serverEventFd: config.ServerEventFd,
		uiEventFd:     config.GuiEventFd,
		guiProcess:    cmd.Process,
		guiDispatch:   make(chan *EncodedClientMsg),
	}

	go ipc.dispatchToGUI()
	return ipc, nil
}

func (ipc *IPCstate) dispatchToGUI() {
	for msg := range ipc.guiDispatch {
		offset := GetWriteOffset(ipc.guiBuffer.memory)
		handler := ipc.cmdHandler
		j := int(offset)
		binary.NativeEndian.PutUint32(handler.Buffer[j:], msg.CmdType)
		binary.NativeEndian.PutUint32(handler.Buffer[j+4:], msg.CmdPayloadSize)
		copy(handler.Buffer[j+8:], msg.Payload)
		// GUI listens to server with server event fd to handle events, so we write to server eventfd
		_, err := ipc.serverEventFd.Write(1)
		if err != nil {
			logger.Log.Error("error signal to GUI eventfd", "error", slog.AnyValue(err))
		}
		logger.Log.Info("message sent to GUI", "cmd_type", msg.CmdType, "payload", string(msg.Payload))
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
	for {
		select {
		case encodedClientMsg := <-ipcHandler.Queue:
			eventMsg := events.EventMsg{
				Type: events.ClientCmdEnum(encodedClientMsg.CmdType),
			}

			var rawData struct {
				Err  error           `json:"error"`
				Data json.RawMessage `json:"data"`
			}
			if len(encodedClientMsg.Payload) > 0 {
				logger.Log.Info("received raw data from ipc", "data", string(encodedClientMsg.Payload))
				fmt.Println(encodedClientMsg.Payload)
				err := json.Unmarshal(encodedClientMsg.Payload, &rawData)
				if err != nil {
					logger.Log.Error(err.Error())
					continue
				}
				logger.Log.Info("successfully parsed raw data", "data", rawData)
				err = json.Unmarshal(rawData.Data, &eventMsg)
				if err != nil {
					logger.Log.Error(err.Error())
				}
				logger.Log.Info("successfully raw data into event msg data", "data", eventMsg)
			}

			ipc.outgoing <- &eventMsg
		case netBridgeMsg := <-ipc.ingoing:
			ipc.processBusMsg(netBridgeMsg)
		}

	}
}

func (ipc *IPCstate) processBusMsg(msg *events.EventMsg) {
	var buffer []byte
	var err error
	var packet *events.MemoryProtocolPacket
	logger.Log.Info("received bus message from core", "message", msg)
	packet = events.EventToMemoryProtocolPacket(msg)
	if packet != nil {
		buffer, err = json.Marshal(packet)
		if err != nil {
			logger.Log.Error("error encoding response packet", "error", err.Error())
			return
		}
		logger.Log.Info("successfully encoded packet", "packet", packet)
	}
	if buffer != nil {
		buffer = encodePayload(buffer)
		responseMsg := NewEncodedClientMsg(uint32(msg.Type), buffer)
		ipc.guiDispatch <- responseMsg
	}
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
		logger.Log.Info("decoded message", "cmd_type", msg.CmdType)
		updateSize := getUpdateSize(msg)
		logger.Log.Info("%d", "update_", updateSize)
		UpdateWriteOffset(ipcState.serverBuffer.memory, updateSize)
		ClearQueue(ipcState.serverBuffer.memory, offset, offset+updateSize)
		ipcState.cmdHandler.Queue <- *msg
	}
}

func decodeMessage(memory []byte, offset uint32) (*EncodedClientMsg, error) {
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

	return NewEncodedClientMsg(commandType, messagePayload), nil
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
	case []byte:
		length = len(d)
	}
	logger.Log.Info("encoding message to gui", "message", data)
	buffer := make([]byte, length)
	_, err := binary.Encode(buffer, binary.NativeEndian, data)
	if err != nil {
		logger.Log.Error("error encoding message to gui", "error", err.Error())
	}
	return buffer
}

func getUpdateSize(msg *EncodedClientMsg) uint32 {
	return msg.CmdPayloadSize + uint32(unsafe.Sizeof(msg.CmdType))
}
