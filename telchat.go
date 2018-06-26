package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"strings"
	"github.com/jwenz723/telchat/clientmap"
	"path/filepath"
	log "github.com/sirupsen/logrus"
)

// Used as an example: https://github.com/kljensen/golang-chat
func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		log.WithField("error", err).Fatal("error parsing config.yml")
	}

	// Setup log path to log messages out to
	if l, err := InitLogging(config.LogDirectory, config.LogLevel, config.LogJSON); err != nil {
		log.WithField("error", err).Fatal("error initializing log file")
	} else {
		defer func() {
			if err = l.Close(); err != nil {
				log.WithField("error", err).Fatal("error closing log file")
			}
		}()
	}

	// Contains a reference to all connected clients
	clients := clientmap.NewClientMap()

	// Channel into which the TCP server will push new connections.
	newConnections := make(chan net.Conn)

	// Channel into which we'll push dead connections for removal from clients.
	deadConnections := make(chan net.Conn)

	// Channel into which messages from clients will be pushed to be broadcast to other clients
	messages := make(chan string)

	// Start the TCP server
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Address, config.Port))
	if err != nil {
		log.WithFields(log.Fields{
			"address": config.Address,
			"port": config.Port,
			"error": err,
		}).Fatal("Failed to start TCP server")
	}
	log.WithFields(log.Fields{
		"address": server.Addr(),
	}).Info("TCP server listening for incoming connections")

	// Tell the server to accept connections forever
	// and push new connections into the newConnections channel.
	go AcceptIncomingConnections(server, newConnections)

	for {
		select {

		// Accept new clients
		case conn := <-newConnections:
			go clients.HandleNewConnection(conn, messages, deadConnections)

		// Accept messages from connected clients
		case message := <-messages:
			go clients.BroadcastMessage(message, deadConnections)

			log.WithFields(log.Fields{
				"message":    message,
				"numClients": clients.Length(),
			}).Info("message broadcasted to clients")

		// Remove dead clients
		case conn := <-deadConnections:
			log.WithField("clientName", clients.GetValue(conn)).Info("client disconnected")
			go clients.HandleDisconnect(conn, messages)
		}
	}
}

func AcceptIncomingConnections(server net.Listener, newConnections chan net.Conn) {
	for {
		conn, err := server.Accept()
		if err != nil {
			log.WithField("error", err).Fatal("error accepting connection")
		}
		newConnections <- conn
	}
}

// InitLogging is used to initialize all properties of the logrus
// logging library.
func InitLogging(logDirectory string, logLevel string, jsonOutput bool) (file *os.File, err error) {
	// if LogDirectory is "" then logging will just go to stdout
	if logDirectory != "" {
		if _, err = os.Stat(logDirectory); os.IsNotExist(err) {
			err := os.MkdirAll(logDirectory, 0777)
			if err != nil {
				return nil, err
			}

			// Chmod is needed because the permissions can't be set by the Mkdir function in Linux
			err = os.Chmod(logDirectory, 0777)
			if err != nil {
				return nil, err
			}
		}
		file, err = os.OpenFile(filepath.Join(logDirectory, fmt.Sprintf("%s%s", time.Now().Local().Format("20060102"), ".log")), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		log.SetOutput(file)
	} else {
		// Output to stdout instead of the default stderr
		log.SetOutput(os.Stdout)
	}

	logLevel = strings.ToLower(logLevel)

	if jsonOutput {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	}

	l, err := log.ParseLevel(logLevel)
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(l)
	}

	return file, nil
}