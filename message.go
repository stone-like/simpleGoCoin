package simplegocoin

import (
	"encoding/json"
	"errors"
)

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
	MSG_NEW_TRANSACTION
	MSG_NEW_BLOCK
	MSG_REQUEST_FULL_CHAIN
	RSP_FULL_CHAIN
	MSG_ENHANCED
)

type Message struct {
	Protocol    string
	Version     string
	MessageType MessageType
	Port        string
	Payload     []byte
}

type Payload interface {
	Unmarshal(content []byte) (Payload, error)
	Marshal() ([]byte, error)
}

type CoreListPayload struct {
	List map[string]struct{}
}

func (c *CoreListPayload) Unmarshal(content []byte) (Payload, error) {
	err := json.Unmarshal(content, c)
	return c, err
}
func (c *CoreListPayload) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

type EnhancedPayload struct {
	Content string
}

func (e *EnhancedPayload) Unmarshal(content []byte) (Payload, error) {
	err := json.Unmarshal(content, e)
	return e, err
}
func (e *EnhancedPayload) Marshal() ([]byte, error) {
	return json.Marshal(e)
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
