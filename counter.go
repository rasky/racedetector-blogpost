// counter.go: simple race detection example
package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

type Server struct {
	conn       net.Listener
	numClients int
}

// NewServer creates a new Server that will listen on the specified proot/addr combo.
// See net.Dial for documentation on proto and addr.
func NewServer(proto, addr string) (*Server, error) {
	conn, err := net.Listen(proto, addr)
	if err != nil {
		return nil, err
	}

	return &Server{conn: conn}, nil
}

// Serve makes Server listen for incoming connection, and spawn a goroutine calling handleClient
// for each new connection.
func (srv *Server) Serve() {
	for {
		conn, err := srv.conn.Accept()
		if err != nil {
			log.Print(err)
			return
		}

		srv.numClients += 1
		go srv.handleClient(conn)
	}
}

func (srv *Server) Close() {
	srv.conn.Close()
}

// handleClient manages the communication with a single client.
// In this example, we just send a predefined message and close the door
func (srv *Server) handleClient(conn net.Conn) {
	io.WriteString(conn, fmt.Sprintf("Ciao, sei il client #%d che si connette a me\n", srv.numClients))
	conn.Close()
}

func main() {
	srv, err := NewServer("tcp", "localhost:2380")
	if err != nil {
		log.Fatal(err)
	}

	srv.Serve()
}
