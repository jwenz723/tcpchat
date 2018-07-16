package tcp

import (
	"github.com/sirupsen/logrus"
	"fmt"
	"net"
)

// Handler contains options for a net.Listener as well as a way to handle all new connections that are accepted
type Handler struct {
	address 			string
	done				chan struct{}
	listener			net.Listener
	logger 				*logrus.Logger
	newConnections 		chan net.Conn
	port 				int
}

// New will create a new Handler for starting a new TCP listener
func New(address string, port int, newConnections chan net.Conn, logger *logrus.Logger) (*Handler, error) {
	return &Handler{
		address:		address,
		done:			make(chan struct{}),
		logger:      	logger,
		newConnections: newConnections,
		port:			port,
	}, nil
}

// Start starts the TCP listener and accepts incoming connections indefinitely until Stop() is called
func (h *Handler) Start() error {
	// Start the TCP listener
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.address, h.port))
	if err != nil {
		return err
	}
	h.listener = l
	h.logger.WithFields(logrus.Fields{
		"address": h.listener.Addr(),
	}).Info("TCP listener accepting connections")

	// pulled this code from the example at: https://stackoverflow.com/a/18969608/3703667
	for {
		type accepted struct {
			conn net.Conn
			err  error
		}
		c := make(chan accepted, 1)
		go func() {
			conn, err := h.listener.Accept()
			c <- accepted{conn, err}
		}()

		select {
		case a := <-c:
			if a.err != nil {
				h.logger.WithField("error", a.err).Fatal("error accepting connection")
				continue
			}
			h.newConnections <- a.conn
		case <-h.done:
			h.logger.Info("stopping TCP listener...")
			return nil
		}
	}
}

// Stop will shutdown the TCP listener
func (h *Handler) Stop() {
	if h.done != nil {
		close(h.done)
	}

	if h.listener != nil {
		h.listener.Close()
	}
}