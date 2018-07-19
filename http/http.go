package http

import (
	"github.com/sirupsen/logrus"
	"github.com/jwenz723/telchat/transporter"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"encoding/json"
	"fmt"
	"net"
)

// Handler serves the HTTP endpoints of the listener
type Handler struct {
	address 			string
	done				chan struct{}
	logger 				*logrus.Logger
	messages 			chan transporter.Message
	newConnections 		chan net.Conn
	port 				int
	router 				*httprouter.Router
}

// New initializes a new http Handler
func New(address string, port int, messages chan transporter.Message, newConnections chan net.Conn, logger *logrus.Logger) *Handler {
	h := &Handler{
		address:		address,
		done:			make(chan struct{}),
		logger:      	logger,
		messages: 	 	messages,
		newConnections: newConnections,
		port:			port,
		router: 		httprouter.New(),
	}

	h.router.POST("/message", h.message)

	return h
}

// Start will start the http listener
func (h *Handler) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.address, h.port))
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- http.Serve(listener, h.router)
	}()

	h.logger.WithFields(logrus.Fields{
		"address": listener.Addr(),
	}).Info("HTTP listener accepting connections")


	for {
		select {
		case e := <-errCh:
			return e
		case <-h.done:
			h.logger.Info("stopping http listener...")
			listener.Close()
			return nil
		}
	}
}

// Stop will shutdown the HTTP listener
func (h *Handler) Stop() {
	if h.done != nil {
		close(h.done)
	}
}

// message is a handler for the /messages endpoint used to send all incoming messages to the h.messages channel
func (h *Handler) message(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dec := json.NewDecoder(r.Body)
	var m transporter.Message
	err := dec.Decode(&m)
	if err != nil {
		panic(err)
	}

	h.messages <- m
	fmt.Fprintln(w, "sent")
	h.logger.WithFields(logrus.Fields{
		"message": m.Message,
		"sender": m.Sender,
	}).Info("received message via http POST")
}
