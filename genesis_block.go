package simplegocoin

import "encoding/json"

type GenesisBlock struct {
	Block
}

func NewGenesisBlock() *GenesisBlock {
	return &GenesisBlock{}
}

func (g *GenesisBlock) Unmarshal(content []byte) (*GenesisBlock, error) {
	err := json.Unmarshal(content, g)
	return g, err
}
func (g *GenesisBlock) Marshal() ([]byte, error) {
	return json.Marshal(g)
}
