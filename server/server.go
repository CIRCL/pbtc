// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

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
	wg   *sync.WaitGroup
	sig  chan struct{}
	host string
	log  adaptor.Log
	mgr  adaptor.Manager
}

func New(options ...func(*Server)) (*Server, error) {
	server := &Server{
		wg:  &sync.WaitGroup{},
		sig: make(chan struct{}),
	}

	for _, option := range options {
		option(server)
	}

	if server.host == "" {
		return nil, errors.New("server: need host address")
	}

	return server, nil
}

func SetHostAddress(host string) func(*Server) {
	return func(server *Server) {
		server.host = host
	}
}

func (server *Server) Start() {
	server.wg.Add(1)
	go server.goListen()
}

func (server *Server) Stop() {
	close(server.sig)
	server.wg.Wait()
}

func (server *Server) SetLog(log adaptor.Log) {
	server.log = log
}

func (server *Server) SetManager(mgr adaptor.Manager) {
	server.mgr = mgr
}

func (server *Server) goListen() {
	defer server.wg.Done()

	ips, ports, err := net.SplitHostPort(server.host)
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
		server.log.Warning("%v: could not listen (%v)", server.host, err)
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
			server.log.Warning("%v: could not accept connection (%v)", server.host, err)
			break
		}

		// we submit the connection for peer creation
		server.mgr.Connection(conn)
	}
}
