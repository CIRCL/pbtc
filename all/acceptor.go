package all

import (
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

	aHandler.connOut = connOut

	go aHandler.handleIPs()
}

func (aHandler *acceptHandler) Stop() {

	close(aHandler.ipIn)
}

func (aHandler *acceptHandler) handleIPs() {

	for IP := range aHandler.ipIn {

		_, ok := aHandler.ipList[IP]
		if ok {
			continue
		}

		addr := net.JoinHostPort(IP, protocolPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}

		aHandler.ipList[IP] = true
		server := NewServer(listener)
		server.Start(aHandler.connOut)
	}
}
