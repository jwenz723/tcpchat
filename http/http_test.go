package http

import (
	"testing"
	"github.com/sirupsen/logrus/hooks/test"
	"net/http"
	"fmt"
	"bytes"
	"time"
	"io/ioutil"
	"encoding/json"
	"net"
	"github.com/jwenz723/telchat/tcp"
)

func TestNew(t *testing.T) {
	address := ""
	port := 8080
	logger, _ := test.NewNullLogger()

	th := tcp.New("", 6000, logger)
	h := New(address, port, th.Messages(), logger)

	if h == nil {
		t.Errorf("received null handler from New()")
	}
}

func TestHandler_Start(t *testing.T) {
	address := ""
	port := 8080
	logger, _ := test.NewNullLogger()
	th := tcp.New("", 6000, logger)
	h := New(address, port, th.Messages(), logger)
	mes := tcp.Message{"in TestHandler_Start()","my name"}
	j, err := json.Marshal(mes)
	if err != nil {
		t.Errorf("failed to marshal Message (%#v) to JSON -> %s", mes, err)
	}

	// Start the http listener
	go func() {
		err = h.Start()
		if err != nil {
			t.Fatalf("failed to start HTTP listener at %s:%d -> %s", address, port, err)
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
	if err == nil {
		conn.Close()
		if !h.Ready {
			t.Errorf("listener at %s:%d is running but h.Ready is not set to true", address, port)
		}

		if h.done == nil {
			t.Errorf("listener at %s:%d is running but h.done is set to nil", address, port)
		}
	}

	if h.Ready {
		// Test POSTing a Message
		a := fmt.Sprintf("http://%s:%d/message", address, port)
		timeout := time.Duration(5 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Post(a, "application/json", bytes.NewBuffer(j))
		if err != nil {
			t.Fatalf("failed to POST Message -> %s", err)
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			e := "sent\n"
			if string(body) != e {
				t.Errorf("expected response (%#v) did not match actual response (%#v) from POST %s", e, string(body), a)
			}
		}

		// Test that the HTTP POST above resulted in a Message being sent through the
		// h.messages channel according to the /message handler function
		select {
		case m := <-h.messages:
			// test that the Message received has the expected content
			if m.Sender != mes.Sender {
				t.Errorf("expected Sender (%s) did not match actual Sender (%s) from messages channel", mes.Sender, m.Sender)
			}
			if m.Message != mes.Message {
				t.Errorf("expected Message (%s) did not match actual Message (%s) from messages channel", mes.Message, m.Message)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("failed to receive Message from messages channel")
		}

		h.Stop()
	}
}

func TestHandler_Stop(t *testing.T) {
	address := ""
	port := 8081
	logger, _ := test.NewNullLogger()
	th := tcp.New("", 6000, logger)
	h := New(address, port, th.Messages(), logger)
	mes := tcp.Message{"in TestHandler_Stop()","my name"}
	j, err := json.Marshal(mes)
	if err != nil {
		t.Errorf("failed to marshal Message (%#v) to JSON -> %s", mes, err)
	}

	success := true
	go func() {
		err = h.Start()
		if err != nil {
			success = false
			t.Fatalf("failed to start HTTP listener at %s:%d -> %s", address, port, err)
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
		h.Stop()
	case <-time.After(3 * time.Second):
	}


	if success {
		if !h.Ready {
			// Test that POSTing a Message fails as expected
			timeout := time.Duration(5 * time.Second)
			client := http.Client{
				Timeout: timeout,
			}
			resp, err := client.Post(fmt.Sprintf("http://%s:%d/message", address, port), "application/json", bytes.NewBuffer(j))
			if err == nil {
				defer resp.Body.Close()
				t.Errorf("POSTed Message to %s:%d did not fail as expected after h.Stop() should have stopped the HTTP listener", address, port)
			}
		} else {
			t.Errorf("h.Ready expected value (%v) does not equal actual value (%v)", false, h.Ready)
		}

		if h.done != nil {
			t.Errorf("h.done not set to nil")
		}
	}
}