package main

import (
	"fmt"
	"os"
	"time"
	"strings"
	"github.com/jwenz723/telchat/transporter"
	"path/filepath"
	"github.com/sirupsen/logrus"
	"github.com/oklog/run"
	"github.com/jwenz723/telchat/http"
	"context"
	"github.com/jwenz723/telchat/tcp"
)

// Source of inspiration for a TCP chat app: https://github.com/kljensen/golang-chat
func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		panic(fmt.Errorf("error parsing config.yml: %s", err))
	}

	// setup logging to file
	logger, teardown, err := InitLogging(config.LogDirectory, config.LogLevel, config.LogJSON)
	if err != nil {
		logger.Fatalf("error initializing logger file -> %v\n", err)
	}
	defer func() {
		if err := teardown(); err != nil {
			panic(err)
		}
	}()

	t := transporter.NewTransporter(logger)
	ctxTCP, cancelTCP := context.WithCancel(context.Background())

	ctxHTTP, cancelHTTP := context.WithCancel(context.Background())
	httpHandler := http.New(config.HTTPAddress, config.HTTPPort, t.Messages(), t.NewConnections(), logger)

	// using a run.Group to handle automatic stopping of all components of the application in
	// the event that one of the components experiences an error.
	var g run.Group
	{
		// Transporter to handle message transport
		g.Add(
			func() error {
				return t.StartTransporter()
			},
			func(err error) {
				t.StopTransporter()
			},
		)
	}
	{
		// TCP listener - accepts messages via telnet connection
		g.Add(
			func() error {
				if err := tcp.Start(config.TCPAddress, config.TCPPort,logger, t.NewConnections(), ctxTCP); err != nil {
					return fmt.Errorf("error starting TCP listener: %s", err)
				}
				return nil
			},
			func(err error) {
				cancelTCP()
			},
		)
	}
	{
		// Web listener - accepts messages via REST api
		g.Add(
			func() error {
				if err := httpHandler.Start(ctxHTTP); err != nil {
					return fmt.Errorf("error starting http listener: %s", err)
				}
				return nil
			},
			func(err error) {
				cancelHTTP()
			},
		)
	}
	//{
	//	g.Add(
	//		func() error {
	//			time.Sleep(10 * time.Second)
	//			return fmt.Errorf("10 second timeout error")
	//		},
	//		func(err error) {
	//			// to nothing
	//		},
	//	)
	//}

	if err := g.Run(); err != nil {
		logger.Fatal(err)
	}
}

// InitLogging is used to initialize all properties of the logrus logging library.
func InitLogging(logDirectory string, logLevel string, jsonOutput bool) (logger *logrus.Logger, teardown func() error, err error) {
	logger = logrus.New()
	var file *os.File
	var tearDownDone bool // used to prevent teardown from being executed more than once

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
		logger.Out = file
	} else {
		// Output to stdout instead of the default stderr
		logger.Out = os.Stdout
	}

	if jsonOutput {
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	}

	l, err := logrus.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(l)
	}

	teardown = func() error {
		if !tearDownDone && file != nil {
			if err = file.Close(); err != nil {
				return fmt.Errorf("error closing logger file -> %v", err)
			}
		}

		tearDownDone = true
		return nil
	}

	return logger, teardown, nil
}