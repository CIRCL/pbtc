package all

import (
	"net"
)

type server struct {
	connOut  chan<- net.Conn
	listener net.Listener
}

func NewServer(listener net.Listener) *server {

	server := &server{
		listener: listener,
	}

	return server
}

func (server *server) Start(connOut chan<- net.Conn) {

	server.connOut = connOut

	go server.handleIncoming()
}

func (server server) handleIncoming() {

	for {

		conn, err := server.listener.Accept()
		if err != nil {
			continue
		}

		server.connOut <- conn
	}

}
