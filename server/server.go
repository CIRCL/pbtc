package server

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/CIRCL/pbtc/adaptor"
)

type Server struct {
	wg    *sync.WaitGroup
	hosts []string
	log   adaptor.Log
}

func New(options ...func(*Server)) (*Server, error) {
	server := &Server{
		wg: &sync.WaitGroup{},
	}

	for _, option := range options {
		option(server)
	}

	if len(server.hosts) == 0 {
		return nil, errors.New("missing address list")
	}

	return server, nil
}

func SetAddressList(hosts ...string) func(*Server) {
	return func(server *Server) {
		server.hosts = hosts
	}
}

func SetLog(log adaptor.Log) func(*Server) {
	return func(server *Server) {
		server.log = log
	}
}

func (server *Server) Close() {

}

func (server *Server) goListen(host string) {
	defer server.wg.Done()

	ips, ports, err := net.SplitHostPort(host)
	if err != nil {
		return
	}

	port, err := strconv.ParseInt(ports, 10, 32)
	if err != nil {
		return
	}

	ip := net.ParseIP(ips)
	if ip == nil {
		return
	}

	addr := &net.TCPAddr{IP: ip, Port: int(port)}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		server.log.Warning("%v: could not listen (%v)", host, err)
		return
	}

	for {
		conn, err := listener.AcceptTCP()
		// unfortunately, listener does not follow the convention of returning
		// an io.EOF on closed connection, so we need to find out like this
		if err != nil &&
			strings.Contains(err.Error(), "use of closed network connection") {
			break
		}
		if err != nil {
			server.log.Warning("%v: could not accept connection (%v)", host, err)
			break
		}

		// we are only interested in TCP connections (should never fail)
		addr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			conn.Close()
			break
		}

		// only accept connections to port 8333 for now (for easy counting)
		if addr.Port != 8333 {
			conn.Close()
			break
		}

		// we submit the connection for peer creation
		//connQ <- conn
	}
}
