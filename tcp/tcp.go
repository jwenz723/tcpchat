package tcp

import (
	"github.com/sirupsen/logrus"
	"fmt"
	"net"
	"context"
)

// Start starts the TCP listener
func Start(address string, port int, logger *logrus.Logger, newConnections chan net.Conn, ctx context.Context) error {
	// Start the TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return err
	}
	defer listener.Close()
	logger.WithFields(logrus.Fields{
		"address": listener.Addr(),
	}).Info("TCP listener accepting connections")

	// pulled this code from the example at: https://stackoverflow.com/a/18969608/3703667
	for {
		type accepted struct {
			conn net.Conn
			err  error
		}
		c := make(chan accepted, 1)
		go func() {
			conn, err := listener.Accept()
			c <- accepted{conn, err}
		}()
		select {
		case a := <-c:
			if a.err != nil {
				logger.WithField("error", err).Fatal("error accepting connection")
				continue
			}
			newConnections <- a.conn
		case <-ctx.Done():
			logger.Info("stopping TCP listener...")
			return nil
		}
	}
}