package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
	"github.com/Pallinder/go-randomdata"
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
	allClients := clientmap.NewClientMap()

	// Channel into which the TCP server will push new connections.
	newConnections := make(chan net.Conn)

	// Channel into which we'll push dead connections for removal from allClients.
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
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.WithField("error", err).Fatal("error accepting connection")
			}
			newConnections <- conn
		}
	}()

	for {
		select {

		// Accept new clients
		case conn := <-newConnections:
			allClients.Write(conn, randomdata.SillyName())

			go func(conn net.Conn) {
				allClients.RLock()
				_, err := conn.Write([]byte(fmt.Sprintf("Enter your name (default: %v)\r\n", allClients.GetValue(conn))))
				allClients.RUnlock()
				if err != nil {
					deadConnections <- conn
					return
				}

				reader := bufio.NewReader(conn)
				incoming, err := reader.ReadString('\n')
				if err != nil {
					deadConnections <- conn
					return
				}

				incoming = strings.Replace(incoming, "\n", "", -1)
				incoming = strings.Replace(incoming, "\r", "", -1)
				if incoming != "" {
					allClients.Write(conn, incoming)
				}

				name := allClients.GetValue(conn)
				_, err = conn.Write([]byte(fmt.Sprintf("Welcome to telchat %v\r\n", name)))
				if err != nil {
					deadConnections <- conn
					return
				}

				messages <- fmt.Sprintf("%v joined\r\n", name)

				for {
					incoming, err := reader.ReadString('\n')
					if err != nil {
						break
					}

					messages <- fmt.Sprintf("%v: %s", name, incoming)
				}

				deadConnections <- conn
			}(conn)

		// Accept messages from connected clients
		case message := <-messages:
			ic := make(chan net.Conn)
			go func() {
				for conn := range ic {
					go func(conn net.Conn, message string) {
						message = fmt.Sprintf("%v %v", time.Now().Format("15:04:05"), message)
						_, err := conn.Write([]byte(message))
						if err != nil {
							deadConnections <- conn
						}
					}(conn, message)
				}
			}()

			if err := allClients.IterateMapKeys(ic); err != nil {
				log.WithField("error", err).Fatal("error iterating through allClients")
			}

			log.WithFields(log.Fields{
				"message": message,
				"numClients": allClients.Length(),
			}).Info("message broadcasted to clients")

		// Remove dead clients
		case conn := <-deadConnections:
			go func() {
				n := allClients.GetValue(conn)
				messages <- fmt.Sprintf("%v disconnected\r\n", n)
				log.WithField("clientName", n).Info("client disconnected")
				allClients.DeleteKey(conn)
			}()
		}
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