package bridge

type Bridge struct {
	in  chan *BridgeMessage
	out chan *BridgeMessage
}

func (bridge Bridge) Send(msg *BridgeMessage) {
	bridge.out <- msg
}
func (bridge Bridge) Get() chan *BridgeMessage {
	return bridge.in
}

func InitBridge(source chan *BridgeMessage, destination chan *BridgeMessage) Bridge {
	return Bridge{
		in:  source,
		out: destination,
	}
}

type BridgeMessage struct {
	Type ClientCmdEnum
	Err  error

	SendFileData *SendFileData
	RequestP2P   *string

	PeerID *string
}

type SendFileData struct {
	Data     []byte
	FileName string
}
