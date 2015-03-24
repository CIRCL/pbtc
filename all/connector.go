package all

import (
	"net"

	"github.com/btcsuite/btcd/wire"
)

type connectionHandler struct {
	addrIn  chan string
	connIn  chan net.Conn
	peerIn  chan *peer
	peerOut chan<- *peer
}

func NewConnectionHandler() *connectionHandler {

	addrIn := make(chan string, bufferConnector)
	connIn := make(chan net.Conn, bufferConnector)
	peerIn := make(chan *peer, bufferConnector)

	cHandler := &connectionHandler{
		addrIn: addrIn,
		connIn: connIn,
		peerIn: peerIn,
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

	cHandler.peerOut = peerOut

	go cHandler.handleAddresses()
	go cHandler.handleConnections()
	go cHandler.handlePeers()
}

func (cHandler *connectionHandler) Stop() {

	close(cHandler.addrIn)
	close(cHandler.connIn)
	close(cHandler.peerIn)
}

func (cHandler *connectionHandler) handleAddresses() {

	for addr := range cHandler.addrIn {

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		cHandler.connIn <- conn
	}
}

func (cHandler *connectionHandler) handleConnections() {

	for conn := range cHandler.connIn {

		me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
		if err != nil {
			return
		}

		you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
		if err != nil {
			return
		}

		nonce, err := wire.RandomUint64()
		if err != nil {
			return
		}

		peer := NewPeer(conn, me, you, nonce, protocolNetwork, protocolVersion)
		peer.Start(cHandler.peerIn)
	}
}

func (cHandler *connectionHandler) handlePeers() {

	for peer := range cHandler.peerIn {

		if peer.IsConnected() {
			cHandler.peerOut <- peer
		}
	}
}
