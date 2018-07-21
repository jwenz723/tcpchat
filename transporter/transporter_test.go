package transporter

import (
	"testing"
	"github.com/sirupsen/logrus/hooks/test"
	"time"
	"fmt"
	"regexp"
	"sync"
	"net"
	"bufio"
	"github.com/jwenz723/telchat/tcp"
)

func TestMessage_String(t *testing.T) {
	m := Message{"test", "name"}
	s := m.String()
	e := fmt.Sprintf("%v[0-9]{2} %s: %s", time.Now().Format("15:04:"), m.Sender, m.Message)
	if matched, _ :=regexp.MatchString(e, s); !matched {
		t.Errorf("actual output (%#v) does not match expected pattern (%#v)", s, e)
	}
}

func TestNewTransporter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)

	if tran == nil {
		t.Errorf("received null transporter")
	}
}

func TestTransporter_StartTransporter(t *testing.T) {
	address := ""
	port := 6000
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)
	//messages := tran.Messages()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tran.StartTransporter()
		if err != nil {
			t.Fatalf("failed to start transporter -> %v", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer func() {
			done <-struct {}{}
		}()
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			t.Fatal(err)
		}
		conn2, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			t.Fatal(err)
		}
		o := ReadFromConn(conn2)
		p := "Enter your name.*"
		if b, _ := regexp.MatchString(p, o); !b {
			t.Errorf("expected: %s\nactual: %s\n", p, o)
		}

		if _, err := fmt.Fprintf(conn, "sender\r\n"); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn2, "receiver\r\n"); err != nil {
			t.Fatal(err)
		}
		o = ReadFromConn(conn2)
		p = "Welcome to telchat.*"
		if b, _ := regexp.MatchString(p, o); !b {
			t.Errorf("expected: %s\nactual: %s\n", p, o)
		}

		if _, err := fmt.Fprintf(conn, "my message\r\n"); err != nil {
			t.Fatal(err)
		}

		o = ReadFromConn(conn2)
		p = ".*: Joined.*"
		if b, _ := regexp.MatchString(p, o); !b {
			t.Errorf("expected: %s\nactual: %s\n", p, o)
		}
		o = ReadFromConn(conn2)
		p = ".*: Joined.*"
		if b, _ := regexp.MatchString(p, o); !b {
			t.Errorf("expected: %s\nactual: %s\n", p, o)
		}
		o = ReadFromConn(conn2)
		p = ".*sender: my message.*"
		if b, _ := regexp.MatchString(p, o); !b {
			t.Errorf("expected: %s\nactual: %s\n", p, o)
		}

		conn.Close()
		conn2.Close()
	}()

	tcpHandler := tcp.New(address, port, tran.NewConnections(), logger)
	go tcpHandler.Start()

	<-done
	tran.StopTransporter()
	wg.Wait()
}

func TestTransporter_StopTransporter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := tran.StartTransporter()
		if err != nil {
			t.Fatalf("failed to start transporter -> %v", err)
		}
	}()
	time.Sleep(1 * time.Second)
	tran.StopTransporter()
}

func TestTransporter_DeadConnections(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)
	d := tran.DeadConnections()

	if d == nil {
		t.Errorf("failed to obtain DeadConnections channel")
	}
}

func TestTransporter_Messages(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)
	d := tran.Messages()

	if d == nil {
		t.Errorf("failed to obtain Messages channel")
	}
}

func TestTransporter_NewConnections(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tran := NewTransporter(logger)
	d := tran.NewConnections()

	if d == nil {
		t.Errorf("failed to obtain NewConnections channel")
	}
}

func ReadFromConn(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	incoming, _ := reader.ReadString('\n')
	return incoming
}