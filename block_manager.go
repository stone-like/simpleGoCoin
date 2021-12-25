package simplegocoin

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

type BlockManager struct {
	Chain []*Block
	lock  sync.RWMutex
}

func NewBlockManager() *BlockManager {
	return &BlockManager{}
}

func (b *BlockManager) SetNewBlock(newBlock *Block) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.Chain = append(b.Chain, newBlock)

}

func getHash(block *Block) (string, error) {
	bytes, err := block.Marshal()
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(bytes)

	return hex.EncodeToString(sum[:]), nil
}
