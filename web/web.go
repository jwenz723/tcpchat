package web

import (
	"github.com/sirupsen/logrus"
	"github.com/jwenz723/telchat/transporter"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"encoding/json"
	"fmt"
	"net"
	"context"
)

// Handler serves the HTTP endpoints of the server
type Handler struct {
	logger *logrus.Logger
	messages chan transporter.Message
	newConnections chan net.Conn
	router *httprouter.Router
}

// New initializes a new web Handler.
func New(logger *logrus.Logger, messages chan transporter.Message, newConnections chan net.Conn) *Handler {
	h := &Handler{
		logger:      	logger,
		messages: 	 	messages,
		newConnections: newConnections,
		router: 		httprouter.New(),
	}

	h.router.POST("/message", h.message)

	return h
}

func (h *Handler) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- http.Serve(listener, h.router)
	}()


	for {
		select {
		case e := <-errCh:
			return e
		case <-ctx.Done():
			h.logger.Info("stopping web listener...")
			listener.Close()
			return nil
		}
	}
}

func (h *Handler) message(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dec := json.NewDecoder(r.Body)
	var m transporter.Message
	err := dec.Decode(&m)
	if err != nil {
		panic(err)
	}
	m.Message = fmt.Sprintf("%s\r\n", m.Message)

	h.messages <- m
	fmt.Fprintln(w, "sent")
	h.logger.WithFields(logrus.Fields{
		"message": m.Message,
		"sender": m.Sender,
	}).Info("received message via http POST")
}
