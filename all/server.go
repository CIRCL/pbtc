package all

import (
	"log"
	"net"
)

type server struct {
	addrIn       chan string
	connOut      chan<- net.Conn
	listenerList map[string]net.Listener
}

func NewServer() *server {
	addrIn := make(chan string)
	listenerList := make(map[string]net.Listener)

	svr := &server{
		addrIn:       addrIn,
		listenerList: listenerList,
	}

	return svr
}

func (svr *server) GetAddrIn() chan<- string {
	return svr.addrIn
}

func (svr *server) Start(connOut chan<- net.Conn) {
	svr.connOut = connOut

	go svr.handleAddresses()
}

func (svr *server) Stop() {
	for _, listener := range svr.listenerList {
		listener.Close()
	}

	close(svr.addrIn)
}

func (svr *server) handleAddresses() {
	for addr := range svr.addrIn {
		_, ok := svr.listenerList[addr]
		if ok {
			continue
		}

		log.Println("Creating listener:", addr)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Println("Can't accept connections:", addr, err)
			continue
		}

		log.Println("Accepting connections:", addr)

		go svr.acceptConnections(listener)
		svr.listenerList[addr] = listener
	}
}

func (svr *server) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		svr.connOut <- conn
	}
}
