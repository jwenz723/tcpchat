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
	"github.com/oklog/run"
	"github.com/jwenz723/telchat/web"
	"context"
)

// Channel into which messages from clients will be pushed to be broadcast to other clients
var (
	logger *logrus.Logger
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

	t := transporter.NewTransporter(logger)
	ctxWeb, cancelWeb := context.WithCancel(context.Background())
	webHandler := web.New(logger, t.Messages(), t.NewConnections())

	var g run.Group
	// Transporter to handle message transport
	{
		g.Add(
			func() error {
				return t.StartTransporter()
			},
			func(err error) {
				t.StopTransporter()
			},
		)
	}
	// TCP handler
	{
		done := make(chan struct{})
		g.Add(
			func() error {
				return StartTCPServer(config.Address, config.Port, t.NewConnections(), done)
			},
			func(err error) {
				close(done)
			},
		)
	}
	// Web handler.
	{
		g.Add(
			func() error {
				if err := webHandler.Run(ctxWeb); err != nil {
					return fmt.Errorf("error starting web server: %s", err)
				}
				return nil
			},
			func(err error) {
				cancelWeb()
			},
		)
	}

	if err := g.Run(); err != nil {
		logger.Fatal(err)
	}
}

// StartTCPServer starts a new tcp listener to allow clients to connect to
func StartTCPServer(address string, port int, newConnections chan net.Conn, done chan struct{}) error {
	// Start the TCP server
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return err
	}
	logger.WithFields(logrus.Fields{
		"address": server.Addr(),
	}).Info("TCP server listening for incoming connections")

	connections := make(chan net.Conn)
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				logger.WithField("error", err).Fatal("error accepting connection")
			}
			connections <- conn
		}
	}()

	for {
		select {
		case conn := <-connections:
			newConnections <- conn
		case <-done:
			logger.Info("stopping TCP server...")
			close(connections)
			return nil
		}
	}

	return nil
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