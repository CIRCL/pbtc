package all

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type server struct {
	addrIn       chan string
	connOut      chan<- net.Conn
	sigAddr      chan struct{}
	listenerList map[string]net.Listener
	waitGroup    *sync.WaitGroup
	state        uint32
}

func NewServer() *server {
	svr := &server{
		addrIn:       make(chan string, bufferServerAddress),
		listenerList: make(map[string]net.Listener),
		waitGroup:    &sync.WaitGroup{},
		state:        stateIdle,
	}

	return svr
}

func (svr *server) GetAddrIn() chan<- string {
	return svr.addrIn
}

func (svr *server) Start(connOut chan<- net.Conn) {
	if !atomic.CompareAndSwapUint32(&svr.state, stateIdle, stateRunning) {
		return
	}

	log.Println("[SVR] Starting")

	svr.sigAddr = make(chan struct{}, 1)

	svr.connOut = connOut

	svr.handleAddresses()

	log.Println("[SVR] Started")
}

func (svr *server) Stop() {
	if !atomic.CompareAndSwapUint32(&svr.state, stateRunning, stateIdle) {
		return
	}

	log.Println("[SVR] Stopping")

	for _, listener := range svr.listenerList {
		listener.Close()
	}

	close(svr.sigAddr)

	svr.waitGroup.Wait()

	log.Println("[SVR] Stopped")
}

func (svr *server) handleAddresses() {
	svr.waitGroup.Add(1)

	go func() {
		defer svr.waitGroup.Done()

	AddrLoop:
		for {
			select {
			case _, ok := <-svr.sigAddr:
				if !ok {
					break AddrLoop
				}

			case addr, ok := <-svr.addrIn:
				if !ok {
					break AddrLoop
				}

				_, ok = svr.listenerList[addr]
				if ok {
					continue
				}

				listener, err := net.Listen("tcp", addr)
				if err != nil {
					log.Println("[SVR] Can't create listener:", addr, err)
					continue
				}

				log.Println("[SVR] Accepting connections on:", addr)
				svr.acceptConnections(listener)
				svr.listenerList[addr] = listener
			}
		}
	}()
}

func (svr *server) acceptConnections(listener net.Listener) {
	svr.waitGroup.Add(1)

	go func() {
		defer svr.waitGroup.Done()

		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}

			log.Println("[SVR] Accepted connection on:", listener.Addr, conn.RemoteAddr().String())
			svr.connOut <- conn
		}
	}()
}
