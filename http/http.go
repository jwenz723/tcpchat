package http

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"encoding/json"
	"fmt"
	"net"
	"github.com/jwenz723/telchat/tcp"
)

// Handler serves the HTTP endpoints of the listener
type Handler struct {
	address        string
	done           chan struct{}
	logger         *logrus.Logger
	messages       chan tcp.Message
	port           int
	Ready          bool // Indicates that the http listener is ready to accept connections
	router         *httprouter.Router
	startDone	   func() // a callback that can be defined to do something once Start() has done all its work
}

// New initializes a new http Handler
func New(address string, port int, messages chan tcp.Message, logger *logrus.Logger) *Handler {
	h := &Handler{
		address:		address,
		done:			make(chan struct{}),
		logger:      	logger,
		messages: 	 	messages,
		port:			port,
		router: 		httprouter.New(),
	}

	h.router.POST("/message", h.message)

	return h
}

// Start will start the http listener
func (h *Handler) Start() error {
	defer func() {
		h.Ready = false
		close(h.done)
	}()
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.address, h.port))
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- http.Serve(listener, h.router)
	}()

	h.Ready = true
	h.logger.WithFields(logrus.Fields{
		"address": listener.Addr(),
	}).Info("HTTP listener accepting connections")

	h.startDone()
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
	if h.Ready && h.done != nil {
		h.done <- struct {}{}

		// wait for the done channel to be closed (meaning the Start() func has actually stopped running)
		<-h.done
		h.done = nil
	}
}

// message is a handler for the /messages endpoint used to send all incoming messages to the h.messages channel
func (h *Handler) message(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dec := json.NewDecoder(r.Body)
	var m tcp.Message
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
