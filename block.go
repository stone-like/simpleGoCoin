package simplegocoin

import (
	"encoding/json"
	"time"
)

type Block struct {
	timeStamp     time.Time
	transaction   []Transaction
	previousBlock string
}

func NewBlock() *Block {
	return &Block{}
}

func (b *Block) Unmarshal(content []byte) (*Block, error) {
	err := json.Unmarshal(content, b)
	return b, err
}
func (b *Block) Marshal() ([]byte, error) {
	return json.Marshal(b)
}
