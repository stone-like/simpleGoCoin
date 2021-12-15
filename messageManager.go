package simplegocoin

import "encoding/json"

type MessageManager struct {
}

func NewMessageManeger() *MessageManager {
	return &MessageManager{}
}

func (mm *MessageManager) CreateCoreListPayload(coreNodeList map[string]struct{}) Payload {
	return &CoreListPayload{
		List: coreNodeList,
	}
}

func (mm *MessageManager) GetCoreListPayload(payload []byte) (Payload, error) {
	m := &CoreListPayload{}
	return m.Unmarshal(payload)
}

func (mm *MessageManager) CreateEnhancedPayload(content string) Payload {
	return &EnhancedPayload{
		Content: content,
	}
}

func (mm *MessageManager) GetBlockChainPayload(payload []byte, msgType MessageType) (Payload, error) {
	switch msgType {
	case MSG_ENHANCED:
		m := &EnhancedPayload{}
		return m.Unmarshal(payload)
	default:
		return nil, ErrorUnknownMessage
	}
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
	case MSG_CORE_LIST, MSG_NEW_TRANSACTION, MSG_NEW_BLOCK, RSP_FULL_CHAIN, MSG_ENHANCED:
		return msg.Port, msg.Payload, msg.MessageType, nil
	default:
		return msg.Port, []byte{}, msg.MessageType, nil
	}

}
