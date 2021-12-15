package simplegocoin

import "encoding/json"

type MessageManager struct {
}

func NewMessageManeger() *MessageManager {
	return &MessageManager{}
}

func (mm *MessageManager) CreatePayload(coreNodeList map[string]struct{}) *MessagePayload {
	return &MessagePayload{
		CoreNodeList: coreNodeList,
	}
}

func (mm *MessageManager) GetPayload(payload []byte) (*MessagePayload, error) {
	var p MessagePayload
	err := json.Unmarshal(payload, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (mm *MessageManager) CreateWithoutPayload(port string, message MessageType) ([]byte, error) {
	return mm.Create([]byte{}, port, message)
}

func (mm *MessageManager) Create(payload []byte, port string, message MessageType) ([]byte, error) {
	m := &Message{
		Protocol:    PROTOCOL_NAME,
		Version:     MY_VERSION,
		MessageType: message,
		Port:        port,
		Payload:     payload,
	}

	return json.Marshal(m)
}

func (mm *MessageManager) Parse(content []byte) (string, []byte, MessageType, error) {

	var msg Message
	err := json.Unmarshal(content, &msg)
	if err != nil {
		return "", []byte{}, 0, err
	}
	err = msg.Validate()
	if err != nil {
		return "", []byte{}, 0, err
	}

	switch msg.MessageType {
	case MSG_CORE_LIST:
		return msg.Port, msg.Payload, msg.MessageType, nil
	default:
		return msg.Port, []byte{}, msg.MessageType, nil
	}
}
