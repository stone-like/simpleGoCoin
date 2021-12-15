package simplegocoin

import "errors"

type MessageType int

var PROTOCOL_NAME = "simple_bitcoin_protocol"

//todo 0.1.以下のバージョンを比較できるようにする
var MY_VERSION = "0.1"

var (
	ErrorProtocolUnmatch = errors.New("protocol unmatch")
	ErrorVersionUnmatch  = errors.New("version unmatch")
	ErrorUnknownMessage  = errors.New("unknown message")
)

const (
	MSG_ADD MessageType = iota + 1
	MSG_REMOVE
	MSG_CORE_LIST
	MSG_REQUEST_CORE_LIST
	MSG_PING
	MSG_ADD_AS_EDGE
	MSG_REMOVE_EDGE
)

type Message struct {
	Protocol    string
	Version     string
	MessageType MessageType
	Port        string
	Payload     []byte
}

type MessagePayload struct {
	CoreNodeList map[string]struct{}
}

func (m Message) Validate() error {
	if m.Protocol != PROTOCOL_NAME {
		return ErrorProtocolUnmatch
	}

	if m.Version > MY_VERSION {
		return ErrorVersionUnmatch
	}

	return nil
}
