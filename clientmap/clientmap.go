package clientmap

import (
	"sync"
	"net"
	"fmt"
	"bufio"
	"strings"
	"github.com/Pallinder/go-randomdata"
	"time"
)

// clientMap is a concurrency-safe map of connections and connection names
type clientMap struct {
	sync.RWMutex
	m map[net.Conn]string
}

// NewClientMap will create a new clientMap
func NewClientMap() clientMap {
	return clientMap{m: make(map[net.Conn]string)}
}

// Length will return the number of keys within c.m
func (c *clientMap) Length() int {
	return len(c.m)
}

// GetValue will retrieve the value corresponding to key within c.m
func (c *clientMap) GetValue (key net.Conn) string {
	c.RLock()
	defer c.RUnlock()
	return c.m[key]
}

// Add will place the key/value pair into c.m
func (c *clientMap) Add(key net.Conn, value string) {
	c.Lock()
	defer c.Unlock()
	c.m[key] = value
}

// DeleteKey will delete the specified key from c.m
func (c *clientMap) DeleteKey (key net.Conn) {
	c.Lock()
	defer c.Unlock()
	delete(c.m, key)
}

// IterateMapKeys will send each key contained in c.m to a returned channel
func (c *clientMap) IterateMapKeys() <-chan net.Conn {
	i := make(chan net.Conn)

	go func() {
		defer func() {
			c.RUnlock()
			close(i)
		}()

		c.RLock()
		for k := range c.m {
			c.RUnlock()
			i <- k
			c.RLock()
		}
	}()

	return i
}

// HandleNewConnection will add conn into c and setup a reader to allow conn to send messages (broadcast to clients)
func (c *clientMap) HandleNewConnection(conn net.Conn, messages chan string, deadConnections chan net.Conn) {
	name := randomdata.SillyName()
	_, err := conn.Write([]byte(fmt.Sprintf("Enter your name (default: %v)\r\n", name)))
	if err != nil {
		deadConnections <- conn
		return
	}

	reader := bufio.NewReader(conn)
	incoming, err := reader.ReadString('\n')
	if err != nil {
		deadConnections <- conn
		return
	}

	incoming = strings.Replace(incoming, "\n", "", -1)
	incoming = strings.Replace(incoming, "\r", "", -1)
	if incoming != "" {
		name = incoming
	}
	c.Add(conn, name)

	_, err = conn.Write([]byte(fmt.Sprintf("Welcome to telchat %v\r\n", name)))
	if err != nil {
		deadConnections <- conn
		return
	}

	messages <- fmt.Sprintf("%v joined\r\n", name)

	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		messages <- fmt.Sprintf("%v: %s", name, incoming)
	}

	deadConnections <- conn
}

// BroadcastMessage will send message to all clients within c
func (c *clientMap) BroadcastMessage(message string, deadConnections chan net.Conn) {
	for conn := range c.IterateMapKeys() {
		go func(conn net.Conn, message string) {
			message = fmt.Sprintf("%v %v", time.Now().Format("15:04:05"), message)
			_, err := conn.Write([]byte(message))
			if err != nil {
				deadConnections <- conn
			}
		}(conn, message)
	}
}

// HandleDisconnect will do all the necessary work for a disconnected client (conn)
func (c *clientMap) HandleDisconnect(conn net.Conn, messages chan string) {
	n := c.GetValue(conn)
	messages <- fmt.Sprintf("%v disconnected\r\n", n)
	c.DeleteKey(conn)
}