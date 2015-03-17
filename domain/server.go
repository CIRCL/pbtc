package domain

import (
	"errors"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listeners []net.Listener
	started   uint32
	shutdown  uint32
}

func NewServer(ips []net.IP) (*Server, error) {
	var listeners []net.Listener

	if len(ips) == 0 {
		return nil, errors.New("No server IPs provided")
	}

	for _, ip := range ips {
		listener, err := net.Listen("tcp4", ip.String()+":8333")
		if err != nil {
			log.Println(err)
			continue
		}

		listeners = append(listeners, listener)
	}

	if len(listeners) == 0 {
		return nil, errors.New("Could not listen on any provided IP")
	}

	server := &Server{
		listeners: listeners,
	}

	return server, nil
}

func (server *Server) Start() {
	if atomic.AddUint32(&server.started, 1) != 1 {
		return
	}

	for _, listener := range server.listeners {
		go server.handleListener(listener)
	}
}

func (server *Server) handleListener(listener net.Listener) {
	for atomic.LoadUint32(&server.shutdown) == 0 {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		server.handleConn(conn)
	}
}

func (server *Server) handleConn(conn net.Conn) {

}
