package all

import (
	"log"
	"net"
	"time"
)

type connectionHandler struct {
	addrIn   chan string
	nodeEx   chan *node
	connIn   chan net.Conn
	connEx   chan net.Conn
	peerIn   chan *peer
	peerOut  chan<- *peer
	ticker   *time.Ticker
	nodeList map[string]*node
}

func NewConnectionHandler() *connectionHandler {

	addrIn := make(chan string, bufferConnector)
	nodeEx := make(chan *node, bufferConnector)
	connIn := make(chan net.Conn, bufferConnector)
	connEx := make(chan net.Conn, bufferConnector)

	nodeList := make(map[string]*node)

	cHandler := &connectionHandler{
		addrIn:   addrIn,
		nodeEx:   nodeEx,
		connIn:   connIn,
		connEx:   connEx,
		nodeList: nodeList,
	}

	return cHandler
}

func (cHandler *connectionHandler) GetAddrIn() chan<- string {
	return cHandler.addrIn
}

func (cHandler *connectionHandler) GetConnIn() chan<- net.Conn {
	return cHandler.connIn
}

func (cHandler *connectionHandler) GetPeerIn() chan<- *peer {
	return cHandler.peerIn
}

func (cHandler *connectionHandler) Start(peerOut chan<- *peer) {

	log.Println("Starting connection handler")

	cHandler.peerOut = peerOut

	period := time.Second / maxConnsPerSec
	cHandler.ticker = time.NewTicker(period)

	go cHandler.handleAddresses()
	go cHandler.handleNodes()
	go cHandler.handleIncoming()
	go cHandler.handleOutgoing()
}

func (cHandler *connectionHandler) Stop() {

	log.Println("Stopping connection handler")

	close(cHandler.addrIn)
	close(cHandler.nodeEx)
	close(cHandler.connIn)
	close(cHandler.connEx)

	cHandler.ticker.Stop()
}

func (cHandler *connectionHandler) handleAddresses() {

	for addr := range cHandler.addrIn {

		node, ok := cHandler.nodeList[addr]
		if !ok {
			node = NewNode(addr)
			cHandler.nodeList[addr] = node
			cHandler.nodeEx <- node
			continue
		}

		go node.Retry(cHandler.nodeEx)
	}
}

func (cHandler *connectionHandler) handleNodes() {

	for node := range cHandler.nodeEx {

		<-cHandler.ticker.C

		go node.Dial(cHandler.addrIn, cHandler.connEx)
	}

}

func (cHandler *connectionHandler) handleIncoming() {

	for conn := range cHandler.connIn {

		peer, err := NewPeer(conn, protocolNetwork, protocolVersion)
		if err != nil {
			cHandler.addrIn <- conn.RemoteAddr().String()
			conn.Close()
			continue
		}

		peer.Start()
		go peer.Handshake(false, cHandler.addrIn, cHandler.peerOut)
	}
}

func (cHandler *connectionHandler) handleOutgoing() {

	for conn := range cHandler.connEx {

		peer, err := NewPeer(conn, protocolNetwork, protocolVersion)
		if err != nil {
			cHandler.addrIn <- conn.RemoteAddr().String()
			conn.Close()
			continue
		}

		peer.Start()
		go peer.Handshake(true, cHandler.addrIn, cHandler.peerOut)
	}
}
