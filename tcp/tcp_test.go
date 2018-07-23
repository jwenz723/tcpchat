package tcp

import (
	"testing"
	"net"
	"github.com/sirupsen/logrus/hooks/test"
	"time"
	"fmt"
	"bufio"
	"regexp"
)

func TestMessage_String(t *testing.T) {
	m := Message{"test", "name"}
	s := m.String()
	e := fmt.Sprintf("%v[0-9]{2} %s: %s", time.Now().Format("15:04:"), m.Sender, m.Message)
	if matched, _ :=regexp.MatchString(e, s); !matched {
		t.Errorf("actual output (%#v) does not match expected pattern (%#v)", s, e)
	}
}

func TestNew(t *testing.T) {
	logger, _ := test.NewNullLogger()
	h := New("", 6000, logger)

	if h == nil {
		t.Errorf("received null handler from New()")
	}
}

func TestHandler_Start(t *testing.T) {
	address := ""
	port := 6000
	logger, _ := test.NewNullLogger()
	h := New(address, port, logger)

	go func() {
		name := "test name"
		name2 := "test name2"
		message := "test message"
		message2 := "test message2"

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)

		// Extract entry message: Enter your name (default: Toothclover)
		incoming, _ := reader.ReadString('\n')

		// Submit a name
		if _, err := fmt.Fprintf(conn, fmt.Sprintf("%s\r\n", name)); err != nil {
			t.Fatal(err)
		}

		// Extract the welcome message
		incoming, _ = reader.ReadString('\n')
		e := fmt.Sprintf("Welcome to telchat %s\r\n", name)
		if incoming != e {
			t.Errorf("did not receive expected welcome message.\n\tExpected: %#v\n\tActual: %#v", e, incoming)
		}

		// Extract the "Joined" message
		incoming, _ = reader.ReadString('\n')
		e = fmt.Sprintf(".*%s: Joined\r\n", name)
		if m, _ := regexp.MatchString(e, incoming); !m {
			t.Errorf("did not receive expected message.\n\tExpected: %#v\n\tActual: %#v", e, incoming)
		}

		// Connect a 2nd client
		conn2, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		reader2 := bufio.NewReader(conn2)

		// Extract entry message: Enter your name (default: Toothclover)
		incoming, _ = reader2.ReadString('\n')

		// Submit a name for conn2
		if _, err := fmt.Fprintf(conn2, fmt.Sprintf("%s\r\n", name2)); err != nil {
			t.Fatal(err)
		}

		// Extract the "Joined" message for conn2 from conn
		incoming, _ = reader.ReadString('\n')
		e = fmt.Sprintf(".*%s: Joined\r\n", name2)
		if m, _ := regexp.MatchString(e, incoming); !m {
			t.Errorf("did not receive expected message.\n\tExpected: %#v\n\tActual: %#v", e, incoming)
		}

		// Submit a message
		if _, err := fmt.Fprintf(conn, fmt.Sprintf("%s\r\n", message)); err != nil {
			t.Fatal(err)
		}

		// Extract the message
		incoming, _ = reader.ReadString('\n')
		e = fmt.Sprintf(".*%s: %s\r\n", name, message)
		if m, _ := regexp.MatchString(e, incoming); !m {
			t.Errorf("did not receive expected message.\n\tExpected: %#v\n\tActual: %#v", e, incoming)
		}

		// Submit a message as conn2
		if _, err := fmt.Fprintf(conn2, fmt.Sprintf("%s\r\n", message2)); err != nil {
			t.Fatal(err)
		}

		// Extract the message that conn2 sent
		incoming, _ = reader.ReadString('\n')
		e = fmt.Sprintf(".*%s: %s\r\n", name2, message2)
		if m, _ := regexp.MatchString(e, incoming); !m {
			t.Errorf("did not receive expected message.\n\tExpected: %#v\n\tActual: %#v", e, incoming)
		}

		h.Stop()
	}()

	err := h.Start()
	if err != nil {
		t.Errorf("failed to start TCP listener at %s:%d -> %s", address, port, err)
	}
}

func TestHandler_Stop(t *testing.T) {
	address := ""
	port := 6001

	logger, _ := test.NewNullLogger()
	h := New(address, port, logger)

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

func TestHandler_Messages(t *testing.T) {
	logger, _ := test.NewNullLogger()
	h := New("", 6000, logger)
	m := h.Messages()

	if m == nil {
		t.Errorf("failed to obtain Messages channel")
	}
}