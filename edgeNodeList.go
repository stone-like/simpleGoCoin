package simplegocoin

import (
	"log"
	"sync"
)

type EdgeNodeList struct {
	list   map[string]struct{}
	logger *log.Logger
	lock   sync.RWMutex
}

func NewEdgeNodeList(logger *log.Logger) *EdgeNodeList {
	return &EdgeNodeList{
		list:   make(map[string]struct{}),
		logger: logger,
	}
}

func (e *EdgeNodeList) Add(addr string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.logger.Printf("adding edge: %s\n", addr)
	e.list[addr] = struct{}{}
	e.logger.Printf("current Edge Node List: %v\n", e.list)

}

func (e *EdgeNodeList) Remove(addr string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.logger.Printf("removeing edge: %s\n", addr)
	if _, ok := e.list[addr]; !ok {
		return
	}
	delete(e.list, addr)
	e.logger.Printf("current Edge Node List: %v\n", e.list)

}

func (e *EdgeNodeList) Overwrite(newList map[string]struct{}) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.logger.Println("Refresh Edge Node List")
	e.list = newList
	e.logger.Printf("current Edge Node List: %v\n", e.list)

}

func (e *EdgeNodeList) CurrentNodes() map[string]struct{} {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.list
}
