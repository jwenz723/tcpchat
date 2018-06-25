package clientmap

import (
	"sync"
	"net"
)

type clientMap struct {
	sync.RWMutex
	m map[net.Conn]string
}

func NewClientMap() clientMap {
	return clientMap{m: make(map[net.Conn]string)}
}

func (c *clientMap) Length() int {
	return len(c.m)
}

func (c *clientMap) GetValue (key net.Conn) string {
	c.RLock()
	defer c.RUnlock()
	return c.m[key]
}

func (c *clientMap) Write (key net.Conn, value string) {
	c.Lock()
	defer c.Unlock()
	c.m[key] = value
}

func (c *clientMap) DeleteKey (key net.Conn) {
	c.Lock()
	defer c.Unlock()
	delete(c.m, key)
}

func (c *clientMap) IterateMapKeys(iteratorChannel chan net.Conn) error {
	c.RLock()
	defer c.RUnlock()
	for k, _ := range c.m {
		c.RUnlock()
		iteratorChannel <- k
		c.RLock()
	}
	close(iteratorChannel)
	return nil
}