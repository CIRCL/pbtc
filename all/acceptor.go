package all

import (
	"log"
	"net"
)

type acceptHandler struct {
	ipIn    chan string
	connOut chan<- net.Conn
	ipList  map[string]bool
}

func NewAcceptHandler() *acceptHandler {

	ipIn := make(chan string, bufferAcceptor)
	ipList := make(map[string]bool)

	aHandler := &acceptHandler{
		ipIn:   ipIn,
		ipList: ipList,
	}

	return aHandler
}

func (aHandler *acceptHandler) GetIpIn() chan<- string {
	return aHandler.ipIn
}

func (aHandler *acceptHandler) Start(connOut chan<- net.Conn) {

	log.Println("Starting accept handler")

	aHandler.connOut = connOut

	go aHandler.handleIPs()
}

func (aHandler *acceptHandler) Stop() {

	log.Println("Stopping accept handler")

	close(aHandler.ipIn)
}

func (aHandler *acceptHandler) handleIPs() {

	for ip := range aHandler.ipIn {

		_, ok := aHandler.ipList[ip]
		if ok {
			log.Println("Already listening:", ip)
			continue
		}

		addr := net.JoinHostPort(ip, protocolPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Println("Could not create listener:", addr, err)
			continue
		}

		log.Println("Listening on address:", addr)

		aHandler.ipList[ip] = true
		server := NewServer(listener)
		server.Start(aHandler.connOut)
	}
}
