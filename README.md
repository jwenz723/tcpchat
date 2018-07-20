# telchat
golang chat app using telnet (TCP)

### Configuring
Create a file in the same directory as the telchat application called config.yml
with all desired configuration properties. For an example see [config.yml.example](config.yml.example).

### Running
1. Compile the application for your desired architecture and platform:
```
GOOS=<OS> # optional
GOARCH=<Arch> # optional
go build
```
2. Run the application: `./telchat`

### Connecting
Connect a client to the TCP chat server by running:
`telnet <TCPAddress> <TCPPort>`

### Sending Messages Via HTTP
You can send messages via HTTP into the chat server using an HTTP POST to 
http://<HTTPAddress>:<HTTPPort>/message with a JSON payload matching the following format:
```json
{
  "sender":"my name",
  "message":"my message"
}
```

Here is an example of how to send a message using curl:
```
curl -X POST http://localhost:8080/message -d "{\"sender\":\"curler\",\"message\":\"hi\"}"
```