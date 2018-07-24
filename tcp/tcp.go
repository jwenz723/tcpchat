package tcp

import (
	"github.com/sirupsen/logrus"
	"fmt"
	"net"
	"strings"
	"time"
	"sync"
	"github.com/Pallinder/go-randomdata"
	"bufio"
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

// Handler contains options for a net.Listener as well as a way to handle all new connections that are accepted
type Handler struct {
	address 			string
	clients         	map[net.Conn]string
	deadConnections 	chan net.Conn
	done				chan struct{}
	logger 				*logrus.Logger
	messages        	chan Message
	mutex           	*sync.RWMutex
	newConnections 		chan net.Conn
	port 				int
	Ready				bool // Indicates that the http listener is ready to accept connections
}

// New will create a new Handler for starting a new TCP listener
func New(address string, port int, logger *logrus.Logger) *Handler {
	return &Handler{
		address:			address,
		clients:         	make(map[net.Conn]string),
		deadConnections: 	make(chan net.Conn, 1),
		done:				make(chan struct{}),
		logger:      		logger,
		messages:        	make(chan Message, 1),
		mutex:           	&sync.RWMutex{},
		newConnections: 	make(chan net.Conn, 1),
		port:				port,
	}
}

// Start starts the TCP listener and accepts incoming connections indefinitely until Stop() is called
func (h *Handler) Start() error {
	defer func() {
		h.Ready = false
		close(h.done)
	}()

	// Start the TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.address, h.port))
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				h.logger.WithField("error", err).Error("error accepting connection")
			} else {
				h.newConnections <- conn
			}
		}
	}()
	h.Ready = true
	h.logger.WithFields(logrus.Fields{
		"address": listener.Addr(),
	}).Info("TCP listener accepting connections")

	for {
		select {
		// Accept new clients
		case conn := <-h.newConnections:
			go h.handleConnect(conn, h.messages, h.deadConnections)

		// Accept messages from connected clients
		case message := <-h.messages:
			go h.broadcastMessage(message, h.deadConnections)

		// Remove dead clients
		case conn := <-h.deadConnections:
			go h.handleDisconnect(conn, h.messages)

		case <-h.done:
			h.logger.Info("stopping TCP listener...")
			err := listener.Close()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

// Stop will shutdown the TCP listener
func (h *Handler) Stop() {
	for {
		time.Sleep(1 * time.Millisecond)
		if h.Ready && h.done != nil {
			h.done <- struct {}{}

			// wait for the done channel to be closed (meaning the Start() func has actually stopped running)
			<-h.done
			h.done = nil
			return
		}
	}
}

// addClient will place the connection/name (key/value) pair into c.clients
func (h *Handler) addClient(key net.Conn, value string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.clients[key] = value
}

// broadcastMessage will send message to all clients within c
func (h *Handler) broadcastMessage(message Message, deadConnections chan net.Conn) {
	wg := sync.WaitGroup{}
	for conn := range h.iterateClients() {
		wg.Add(1)
		go func(conn net.Conn, message Message) {
			defer wg.Done()
			_, err := conn.Write([]byte(message.String()))
			if err != nil {
				deadConnections <- conn
			}

			h.logger.WithFields(logrus.Fields{
				"message":  message.Message,
				"receiver": h.getClientName(conn),
				"sender":   message.Sender,
			}).Debug("sent message")
		}(conn, message)
	}

	wg.Wait()
	h.logger.WithFields(logrus.Fields{
		"message":    message.Message,
		"numClients": h.numClients(),
		"sender":     message.Sender,
	}).Info("sent message to all clients")
}

// deleteClient will delete the specified key from c.clients
func (h *Handler) deleteClient(key net.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.clients, key)
}

// getClientName will retrieve the name corresponding to specifed client within c.clients or "" if client doesn't exist
func (h *Handler) getClientName(client net.Conn) string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	val, ok := h.clients[client]
	if !ok {
		return ""
	}
	return val
}

// handleConnect will add conn into c and setup a reader to allow conn to send messages (broadcast to clients)
func (h *Handler) handleConnect(conn net.Conn, messages chan Message, deadConnections chan net.Conn) {
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
	h.addClient(conn, name)

	h.logger.WithFields(logrus.Fields{
		"address.local": conn.LocalAddr(),
		"address.remote": conn.RemoteAddr(),
		"name": name,
	}).Info("client connected")

	_, err = conn.Write([]byte(fmt.Sprintf("Welcome to telchat %v\r\n", name)))
	if err != nil {
		deadConnections <- conn
		return
	}

	go func() {
		messages <- Message{"Joined\r\n", name}
	}()

	for {
		m, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		h.logger.WithFields(logrus.Fields{
			"message": m,
			"sender":  name,
		}).Info("received message via tcp")
		messages <- Message{m, name}
	}

	deadConnections <- conn
}

// handleDisconnect will do all the necessary work for a disconnected client (conn)
func (h *Handler) handleDisconnect(conn net.Conn, messages chan Message) {
	h.logger.WithFields(logrus.Fields{
		"address.local":  conn.LocalAddr(),
		"address.remote": conn.RemoteAddr(),
		"name":           h.getClientName(conn),
	}).Info("client disconnected")

	n := h.getClientName(conn)
	messages <- Message{"Disconnected", n}
	h.deleteClient(conn)
}

// iterateClients will send each key contained in c.clients to a returned channel
func (h *Handler) iterateClients() <-chan net.Conn {
	i := make(chan net.Conn)

	go func() {
		defer func() {
			h.mutex.RUnlock()
			close(i)
		}()

		h.mutex.RLock()
		for k := range h.clients {
			h.mutex.RUnlock()
			i <- k
			h.mutex.RLock()
		}
	}()

	return i
}

// numClients will return the number of keys within h.clients
func (h *Handler) numClients() int {
	return len(h.clients)
}

// Messages will return a reference to a channel that all client messages are sent on
func (h *Handler) Messages() chan Message {
	return h.messages
}