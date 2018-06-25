package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"github.com/Pallinder/go-randomdata"
	"strings"
	"github.com/jwenz723/telchat/clientmap"
)



// Used https://github.com/kljensen/golang-chat as an example
func main() {
	// Contains a reference to all connected clients
	allClients := clientmap.NewClientMap()

	// Channel into which the TCP server will push new connections.
	newConnections := make(chan net.Conn)

	// Channel into which we'll push dead connections for removal from allClients.
	deadConnections := make(chan net.Conn)

	// Channel into which we'll push messages from
	// connected clients so that we can broadcast them
	// to every connection in allClients.
	messages := make(chan string, 1)

	// Start the TCP server
	server, err := net.Listen("tcp", ":6000")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Tell the server to accept connections forever
	// and push new connections into the newConnections channel.
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
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

				m := fmt.Sprintf("%v %v joined\r\n", time.Now().Format("15:04:05"), name)
				messages <- m
				log.Printf(m)

				for {
					incoming, err := reader.ReadString('\n')
					if err != nil {
						break
					}

					messages <- fmt.Sprintf("%v %v: %s", time.Now().Format("15:04:05"), name, incoming)
				}

				deadConnections <- conn
			}(conn)

		// Accept messages from connected clients
		case message := <-messages:
			ic := make(chan net.Conn)
			go func() {
				for conn := range ic {
					go func(conn net.Conn, message string) {
						_, err := conn.Write([]byte(message))

						// If there was an error communicating
						// with them, the connection is dead.
						if err != nil {
							deadConnections <- conn
						}
					}(conn, message)
				}
			}()

			if err := allClients.IterateMapKeys(ic); err != nil {
				fmt.Println(err)
			}

			log.Printf("New message broadcast to %d clients: %s", allClients.Length(), message)

		// Remove dead clients
		case conn := <-deadConnections:
			m := fmt.Sprintf("%v %v disconnected\r\n", time.Now().Format("15:04:05"), allClients.GetValue(conn))
			messages <- m
			log.Printf(m)
			allClients.DeleteKey(conn)
		}
	}
}
