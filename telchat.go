package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"strings"
	"github.com/jwenz723/telchat/transporter"
	"path/filepath"
	"github.com/sirupsen/logrus"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"encoding/json"
)

// Channel into which messages from clients will be pushed to be broadcast to other clients
var (
	logger *logrus.Logger
	messages = make(chan transporter.Message)
)

// Used as an example: https://github.com/kljensen/golang-chat
func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		logger.WithField("error", err).Fatal("error parsing config.yml")
	}

	// setup logging to file
	l, teardown, err := InitLogging(config.LogDirectory, config.LogLevel, config.LogJSON)
	if err != nil {
		logger.Fatalf("error initializing logger file -> %v\n", err)
	}
	defer teardown()
	logger = l

	// Contains a reference to all connected clients
	clients := transporter.NewClientMap(logger)

	// Channel into which the TCP server will push new connections.
	newConnections := make(chan net.Conn)

	// Channel into which we'll push dead connections for removal from clients.
	deadConnections := make(chan net.Conn)



	// Tell the server to accept connections forever
	// and push new connections into the newConnections channel.
	go StartTCPServer(config.Address, config.Port, newConnections)

	go func() {
		for {
			select {
			// Accept new clients
			case conn := <-newConnections:
				go clients.HandleConnect(conn, messages, deadConnections)

			// Accept messages from connected clients
			case message := <-messages:
				go clients.BroadcastMessage(message, deadConnections)

			// Remove dead clients
			case conn := <-deadConnections:
				go clients.HandleDisconnect(conn, messages)
			}
		}
	}()

	router := httprouter.New()
	router.POST("/message", HandleMessage)

	logger.Fatal(http.ListenAndServe(":8080", router))
}

// StartTCPServer starts a new tcp listener to allow clients to connect to
func StartTCPServer(address string, port int, newConnections chan net.Conn) {
	// Start the TCP server
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"address": address,
			"port": port,
			"error": err,
		}).Fatal("Failed to start TCP server")
	}
	logger.WithFields(logrus.Fields{
		"address": server.Addr(),
	}).Info("TCP server listening for incoming connections")

	for {
		conn, err := server.Accept()
		if err != nil {
			logger.WithField("error", err).Fatal("error accepting connection")
		}
		newConnections <- conn
	}
}

// InitLogging is used to initialize all properties of the logrus
// logging library.
func InitLogging(logDirectory string, logLevel string, jsonOutput bool) (logger *logrus.Logger, teardown func(), err error) {
	logger = logrus.New()
	var file *os.File

	// if LogDirectory is "" then logging will just go to stdout
	if logDirectory != "" {
		if _, err = os.Stat(logDirectory); os.IsNotExist(err) {
			err := os.MkdirAll(logDirectory, 0777)
			if err != nil {
				return nil, nil, err
			}

			// Chmod is needed because the permissions can't be set by the Mkdir function in Linux
			err = os.Chmod(logDirectory, 0777)
			if err != nil {
				return nil, nil, err
			}
		}
		file, err = os.OpenFile(filepath.Join(logDirectory, fmt.Sprintf("%s%s", time.Now().Local().Format("20060102"), ".log")), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		//logger.SetOutput(file)
		logger.Out = file
	} else {
		// Output to stdout instead of the default stderr
		//logrus.SetOutput(os.Stdout)
		logger.Out = os.Stdout
	}

	if jsonOutput {
		//logger.SetFormatter(&logrus.JSONFormatter{})
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		//logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	}

	l, err := logrus.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(l)
	}

	teardown = func() {
		if err = file.Close(); err != nil {
			logger.Errorf("error closing logger file -> %v\n", err)
		}
	}

	return logger, teardown, nil
}

func HandleMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	dec := json.NewDecoder(r.Body)
	var m transporter.Message
	err := dec.Decode(&m)
	if err != nil {
		panic(err)
	}
	m.Message = fmt.Sprintf("%s\r\n", m.Message)

	messages <- m
	fmt.Fprintln(w, "sent")
	logger.WithFields(logrus.Fields{
		"message": m.Message,
		"sender": m.Sender,
	}).Info("received message via http POST")
}

