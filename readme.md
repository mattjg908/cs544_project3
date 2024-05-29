## Simple Go Echo Demo using QUIC

There is a single binary that is used to run both the client and the server

- `server`:
  - to start a server:
    - `go run cmd/echo/echo.go -server`
  - to start a second server, use a different port number (and/or server-ip):
    - `go run cmd/echo/echo.go -server -port=4243`

- `client`:
  - to connect:
    - `go run cmd/echo/echo.go -client -mtype=connect -data="some_nickname|password123"`
  - to connect to a different server, pass that server's port number (and/or server-ip):
    - `go run cmd/echo/echo.go -client -mtype=connect -data="bar|password123" -port=4243`
  - to message another client:
    - a second client must connect, then
    - `go run cmd/echo/echo.go -client -mtype=connect -data="some_nickname|hello some_nickname"`
  - to list connected clients:
    - after connecting, `list`
  - to toggle being away
    - after connecting, `away`
      - this marks you as "away", the app will auto-respond to messages
    - typing `away` again will unmark you as away and will stop auto-responding
- `help on all flags`: `go run cmd/echo/echo.go -help`

The server will wait for a connection, just a simple echo.  This solution uses goroutines and is concurrent.

There is also a pdu defined in the `pkg/pdu` package

Solution derived from the excellent work of the `quic-go` team based on the example: https://github.com/quic-go/quic-go/tree/master/example/echo