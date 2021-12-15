package simplegocoin

import (
	"log"
	"sync"
)

//CoreNodeListは競合するときにgoroutineセーフじゃないのでここでロックをかけて処理

type CoreNodeList struct {
	list   map[string]struct{}
	logger *log.Logger
	lock   sync.RWMutex
}

func NewCoreNodeList(logger *log.Logger) *CoreNodeList {
	return &CoreNodeList{
		list:   make(map[string]struct{}),
		logger: logger,
	}
}

func (c *CoreNodeList) Add(addr string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Printf("adding peer: %s\n", addr)
	c.list[addr] = struct{}{}
	c.logger.Printf("current Core Node List: %v\n", c.list)

}

func (c *CoreNodeList) Remove(addr string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Printf("removeing peer: %s\n", addr)
	if _, ok := c.list[addr]; !ok {
		return
	}
	delete(c.list, addr)
	c.logger.Printf("current Core Node List: %v\n", c.list)

}

func (c *CoreNodeList) Overwrite(newList map[string]struct{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Println("Refresh Core Node List")
	c.list = newList
	c.logger.Printf("current Core Node List: %v\n", c.list)

}

func (c *CoreNodeList) CurrentNodes() map[string]struct{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.list
}
