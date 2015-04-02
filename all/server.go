package all

import (
	"log"
	"net"
	"sync"
)

type server struct {
	addrIn       chan string
	connOut      chan<- net.Conn
	listenerList map[string]net.Listener
	waitGroup    *sync.WaitGroup
}

func NewServer() *server {
	addrIn := make(chan string, bufferServerAddress)
	listenerList := make(map[string]net.Listener)

	svr := &server{
		addrIn:       addrIn,
		listenerList: listenerList,
		waitGroup:    &sync.WaitGroup{},
	}

	return svr
}

func (svr *server) GetAddrIn() chan<- string {
	return svr.addrIn
}

func (svr *server) Start(connOut chan<- net.Conn) {
	svr.connOut = connOut

	svr.waitGroup.Add(1)
	go svr.handleAddresses()
}

func (svr *server) Stop() {
	svr.waitGroup.Wait()
}

func (svr *server) handleAddresses() {
	for addr := range svr.addrIn {
		_, ok := svr.listenerList[addr]
		if ok {
			continue
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Println("Can't create listener:", addr, err)
			continue
		}

		log.Println("Accepting connections on:", addr)
		svr.waitGroup.Add(1)
		go svr.acceptConnections(listener)
		svr.listenerList[addr] = listener
	}

	for _, listener := range svr.listenerList {
		listener.Close()
	}

	svr.waitGroup.Done()
}

func (svr *server) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		log.Println("Accepted connection on:", listener.Addr, conn.RemoteAddr().String())
		svr.connOut <- conn
	}

	svr.waitGroup.Done()
}
