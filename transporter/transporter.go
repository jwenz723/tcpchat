package transporter

import (
	"sync"
	"net"
	"fmt"
	"bufio"
	"strings"
	"github.com/Pallinder/go-randomdata"
	"time"
	"github.com/sirupsen/logrus"
)

// Message is to be broadcasted to clients
type Message struct {
	Message string `json:"message"`
	Sender string `json:"sender"`
}

// String converts m into a message that can be displayed to a user
func (m *Message) String() string {
	return fmt.Sprintf("%v %s: %s", time.Now().Format("15:04:05"), m.Sender, m.Message)
}

// transporter is a concurrency-safe map of client connections and client names
type transporter struct {
	sync.RWMutex
	clients map[net.Conn]string
	Logger  *logrus.Logger
}

// NewClientMap will create a new transporter
func NewClientMap(logger *logrus.Logger) transporter {
	return transporter{
		clients: make(map[net.Conn]string),
		Logger:  logger,
	}
}

// AddClient will place the connection/name (key/value) pair into c.clients
func (t *transporter) AddClient(key net.Conn, value string) {
	t.Lock()
	defer t.Unlock()
	t.clients[key] = value
}

// BroadcastMessage will send message to all clients within c
func (t *transporter) BroadcastMessage(message Message, deadConnections chan net.Conn) {
	wg := sync.WaitGroup{}
	for conn := range t.IterateClients() {
		wg.Add(1)
		go func(conn net.Conn, message Message) {
			defer wg.Done()
			_, err := conn.Write([]byte(message.String()))
			if err != nil {
				deadConnections <- conn
			}

			t.Logger.WithFields(logrus.Fields{
				"message":  message.Message,
				"receiver": t.GetClientName(conn),
				"sender":   message.Sender,
			}).Debug("sent message")
		}(conn, message)
	}

	wg.Wait()
	t.Logger.WithFields(logrus.Fields{
		"message":    message.Message,
		"numClients": t.NumClients(),
		"sender":     message.Sender,
	}).Info("sent message to all clients")
}

// DeleteClient will delete the specified key from c.clients
func (t *transporter) DeleteClient(key net.Conn) {
	t.Lock()
	defer t.Unlock()
	delete(t.clients, key)
}

// GetClientName will retrieve the name corresponding to specifed client within c.clients or "" if client doesn't exist
func (t *transporter) GetClientName(client net.Conn) string {
	t.RLock()
	defer t.RUnlock()
	val, ok := t.clients[client]
	if !ok {
		return ""
	}
	return val
}

// HandleConnect will add conn into c and setup a reader to allow conn to send messages (broadcast to clients)
func (t *transporter) HandleConnect(conn net.Conn, messages chan Message, deadConnections chan net.Conn) {
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
	t.AddClient(conn, name)

	t.Logger.WithFields(logrus.Fields{
		"addr": conn.RemoteAddr(),
		"name": name,
	}).Info("client connected")

	_, err = conn.Write([]byte(fmt.Sprintf("Welcome to telchat %v\r\n", name)))
	if err != nil {
		deadConnections <- conn
		return
	}

	messages <- Message{"Joined\r\n", name}

	for {
		m, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		t.Logger.WithFields(logrus.Fields{
			"message": m,
			"sender":  name,
		}).Info("received message via tcp")
		messages <- Message{m, name}
	}

	deadConnections <- conn
}

// HandleDisconnect will do all the necessary work for a disconnected client (conn)
func (t *transporter) HandleDisconnect(conn net.Conn, messages chan Message) {
	t.Logger.WithFields(logrus.Fields{
		"addr": conn.RemoteAddr(),
		"name": t.GetClientName(conn),
	}).Info("client disconnected")

	n := t.GetClientName(conn)
	messages <- Message{"Disconnected", n}
	t.DeleteClient(conn)
}

// IterateClients will send each key contained in c.clients to a returned channel
func (t *transporter) IterateClients() <-chan net.Conn {
	i := make(chan net.Conn)

	go func() {
		defer func() {
			t.RUnlock()
			close(i)
		}()

		t.RLock()
		for k := range t.clients {
			t.RUnlock()
			i <- k
			t.RLock()
		}
	}()

	return i
}

// NumClients will return the number of keys within c.clients
func (t *transporter) NumClients() int {
	return len(t.clients)
}