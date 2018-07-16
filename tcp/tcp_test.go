package tcp

import (
	"testing"
	"net"
	"github.com/sirupsen/logrus/hooks/test"
	"time"
	"fmt"
)

func TestNew(t *testing.T) {
	logger, _ := test.NewNullLogger()
	newConnections := make(chan net.Conn)
	h, err := New("", 6000, newConnections, logger)
	if err != nil {
		t.Errorf("failed to create new handler -> %s", err)
	}

	if h == nil {
		t.Errorf("received null handler from New()")
	}
}

func TestHandler_Start(t *testing.T) {
	logger, _ := test.NewNullLogger()
	newConnections := make(chan net.Conn)
	h, err := New("", 6000, newConnections, logger)

	go func() {
		err = h.Start()
		if err != nil {
			t.Errorf("failed to start TCP listener -> %s", err)
		}
	}()
	defer h.Stop()

	_, err = net.Dial("tcp", ":6000")
	if err != nil {
		t.Errorf("failed to connect via TCP to :6000 -> %s", err)
	}

	// Test that the TCP connection above resulted in a net.Conn being sent on newConnections
	select {
	case <-newConnections:
		// successfully received new connection
	case <-time.After(1 * time.Second):
		t.Errorf("failed to receive new net.Conn from newConnections channel")
	}
}

func TestHandler_Stop(t *testing.T) {
	address := ""
	port := 6000

	logger, _ := test.NewNullLogger()
	newConnections := make(chan net.Conn)
	h, err := New(address, port, newConnections, logger)

	// Test that h.Stop successfully stops the TCP listener
	go func() {
		err = h.Start()
		if err != nil {
			t.Errorf("failed to start TCP listener -> %s", err)
		}
	}()
	time.Sleep(1 * time.Millisecond)
	h.Stop()

	_, err = net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err == nil {
		t.Errorf("connect via TCP to %s:%d after h.Stop() should have stopped the TCP listener -> %s", address, port, err)
	}
}