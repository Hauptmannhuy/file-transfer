package bridge

type ClientCmdEnum uint8

const (
	CmdRequestAddresses ClientCmdEnum = iota + 1
	CmdIdentifyHost
	CmdSendFile
	CmdRequestP2P
	CmdResponseP2P
)

var clientCmdNames = map[ClientCmdEnum]string{
	CmdRequestAddresses: "RequestAddresses",
	CmdIdentifyHost:     "IdentifyHost",
	CmdSendFile:         "SendFile",
	CmdRequestP2P:       "RequestP2P",
	CmdResponseP2P:      "ResponseP2P",
}

func (c ClientCmdEnum) String() string {
	name, ok := clientCmdNames[c]
	if !ok {
		return "unknown command"
	}
	return name
}

func (c ClientCmdEnum) Exists() bool {
	_, ok := clientCmdNames[c]
	return ok
}
