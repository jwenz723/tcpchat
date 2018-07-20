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
	if !strings.HasSuffix(m.Message, "\r\n") {
		m.Message += "\r\n"
	}
	return fmt.Sprintf("%v %s: %s", time.Now().Format("15:04:05"), m.Sender, m.Message)
}

// Transporter is used to send messages between connected clients. Any messages that are sent into the Messages()
// channel will be transported outbound to all connected clients.
type Transporter interface {
	DeadConnections() chan net.Conn
	Messages() chan Message
	NewConnections() chan net.Conn
	StartTransporter() error
	StopTransporter()
}

// NewTransporter will create a new Transporter used to send client Messages to other clients
func NewTransporter(logger *logrus.Logger) Transporter {
	return &transporter{
		clients:         make(map[net.Conn]string),
		deadConnections: make(chan net.Conn, 1),
		done:            make(chan struct{}),
		logger:          logger,
		messages:        make(chan Message, 1),
		mutex:           &sync.RWMutex{},
		newConnections:  make(chan net.Conn, 1),
	}
}

// transporter is a concurrency-safe map of client connections and client names that will broadcast messages to clients
type transporter struct {
	clients         map[net.Conn]string
	deadConnections chan net.Conn
	done            chan struct{}
	logger          *logrus.Logger
	messages        chan Message
	mutex           *sync.RWMutex
	newConnections  chan net.Conn
}

// StartTransporter will start the transporter
func (t *transporter) StartTransporter() error {
	for {
		select {
		// Accept new clients
		case conn := <-t.newConnections:
			go t.handleConnect(conn, t.messages, t.deadConnections)

			// Accept messages from connected clients
		case message := <-t.messages:
			go t.broadcastMessage(message, t.deadConnections)

			// Remove dead clients
		case conn := <-t.deadConnections:
			go t.handleDisconnect(conn, t.messages)

		case <-t.done:
			return nil
		}
	}

	return nil
}

// StopTransporter will stop the transporter
func (t *transporter) StopTransporter() {
	t.logger.Info("stopping transporter...")
	close(t.done)
}

// DeadConnections will return a reference to a channel that all dead client connections are sent on
func (t *transporter) DeadConnections() chan net.Conn {
	return t.deadConnections
}

// Messages will return a reference to a channel that all client messages are sent on
func (t *transporter) Messages() chan Message {
	return t.messages
}

// NewConnections will return a reference to a channel that all new client connections are sent on
func (t *transporter) NewConnections() chan net.Conn {
	return t.newConnections
}

// addClient will place the connection/name (key/value) pair into c.clients
func (t *transporter) addClient(key net.Conn, value string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.clients[key] = value
}

// broadcastMessage will send message to all clients within c
func (t *transporter) broadcastMessage(message Message, deadConnections chan net.Conn) {
	wg := sync.WaitGroup{}
	for conn := range t.iterateClients() {
		wg.Add(1)
		go func(conn net.Conn, message Message) {
			defer wg.Done()
			_, err := conn.Write([]byte(message.String()))
			if err != nil {
				deadConnections <- conn
			}

			t.logger.WithFields(logrus.Fields{
				"message":  message.Message,
				"receiver": t.getClientName(conn),
				"sender":   message.Sender,
			}).Debug("sent message")
		}(conn, message)
	}

	wg.Wait()
	t.logger.WithFields(logrus.Fields{
		"message":    message.Message,
		"numClients": t.numClients(),
		"sender":     message.Sender,
	}).Info("sent message to all clients")
}

// deleteClient will delete the specified key from c.clients
func (t *transporter) deleteClient(key net.Conn) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.clients, key)
}

// getClientName will retrieve the name corresponding to specifed client within c.clients or "" if client doesn't exist
func (t *transporter) getClientName(client net.Conn) string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	val, ok := t.clients[client]
	if !ok {
		return ""
	}
	return val
}

// handleConnect will add conn into c and setup a reader to allow conn to send messages (broadcast to clients)
func (t *transporter) handleConnect(conn net.Conn, messages chan Message, deadConnections chan net.Conn) {
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
	t.addClient(conn, name)

	t.logger.WithFields(logrus.Fields{
		"address.local": conn.LocalAddr(),
		"address.remote": conn.RemoteAddr(),
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

		t.logger.WithFields(logrus.Fields{
			"message": m,
			"sender":  name,
		}).Info("received message via tcp")
		messages <- Message{m, name}
	}

	deadConnections <- conn
}

// handleDisconnect will do all the necessary work for a disconnected client (conn)
func (t *transporter) handleDisconnect(conn net.Conn, messages chan Message) {
	t.logger.WithFields(logrus.Fields{
		"address.local": conn.LocalAddr(),
		"address.remote": conn.RemoteAddr(),
		"name": t.getClientName(conn),
	}).Info("client disconnected")

	n := t.getClientName(conn)
	messages <- Message{"Disconnected", n}
	t.deleteClient(conn)
}

// iterateClients will send each key contained in c.clients to a returned channel
func (t *transporter) iterateClients() <-chan net.Conn {
	i := make(chan net.Conn)

	go func() {
		defer func() {
			t.mutex.RUnlock()
			close(i)
		}()

		t.mutex.RLock()
		for k := range t.clients {
			t.mutex.RUnlock()
			i <- k
			t.mutex.RLock()
		}
	}()

	return i
}

// numClients will return the number of keys within c.clients
func (t *transporter) numClients() int {
	return len(t.clients)
}