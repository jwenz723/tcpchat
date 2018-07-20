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
	h := New("", 6000, newConnections, logger)

	if h == nil {
		t.Errorf("received null handler from New()")
	}
}

func TestHandler_Start(t *testing.T) {
	address := ""
	port := 6000
	logger, _ := test.NewNullLogger()
	newConnections := make(chan net.Conn)
	h := New(address, port, newConnections, logger)

	go func() {
		err := h.Start()
		if err != nil {
			t.Errorf("failed to start TCP listener at %s:%d -> %s", address, port, err)
		}
	}()

	// Wait for h.Start() to do its thing or timeout after a few seconds
	c := make(chan struct{})
	go func() {
		for !h.Ready && h.done != nil {
			time.Sleep(1 * time.Millisecond)
		}
		c <- struct{}{}
	}()
	select {
	case <-c:
	case <-time.After(3 * time.Second):
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		t.Errorf("failed to connect via TCP to %s:%d -> %s", address, port, err)
	} else {
		conn.Close()
		if !h.Ready {
			t.Errorf("listener at %s:%d is running but h.Ready is not set to true", address, port)
		}

		if h.done == nil {
			t.Errorf("listener at %s:%d is running but h.done is set to nil", address, port)
		}
	}

	if h.Ready {
		// Test that the TCP connection above resulted in a net.Conn being sent on newConnections
		select {
		case <-newConnections:
			// successfully received new connection
		case <-time.After(1 * time.Second):
			t.Errorf("failed to receive new net.Conn from newConnections channel")
		}

		h.Stop()
	}
}

func TestHandler_Stop(t *testing.T) {
	address := ""
	port := 6001

	logger, _ := test.NewNullLogger()
	newConnections := make(chan net.Conn)
	h := New(address, port, newConnections, logger)

	// Test that h.Stop successfully stops the TCP listener
	go func() {
		err := h.Start()
		if err != nil {
			t.Errorf("failed to start TCP listener at %s:%d -> %s", address, port, err)
		}
	}()
	// Wait for h.Start() to do its thing or timeout after a few seconds
	c := make(chan struct{})
	go func() {
		for !h.Ready && h.done != nil {
			time.Sleep(1 * time.Millisecond)
		}
		c <- struct{}{}
	}()
	select {
	case <-c:
	case <-time.After(3 * time.Second):
	}
	h.Stop()

	if h.done != nil {
		t.Errorf("h.done not set to nil")
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err == nil {
		t.Errorf("connected via TCP to %s:%d after h.Stop() should have stopped the TCP listener", address, port)
	}
	if conn != nil {
		conn.Close()
	}
}