package all

import (
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type connectionHandler struct {
	addrIn   chan string
	connIn   chan net.Conn
	peerIn   chan *peer
	peerOut  chan<- *peer
	ticker   *time.Ticker
	numConns uint32
}

func NewConnectionHandler() *connectionHandler {

	addrIn := make(chan string, bufferConnector)
	connIn := make(chan net.Conn, bufferConnector)
	peerIn := make(chan *peer, bufferConnector)

	cHandler := &connectionHandler{
		addrIn:   addrIn,
		connIn:   connIn,
		peerIn:   peerIn,
		numConns: 0,
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
	go cHandler.handleConnections()
	go cHandler.handlePeers()
}

func (cHandler *connectionHandler) Stop() {

	log.Println("Stopping connection handler")

	cHandler.ticker.Stop()

	close(cHandler.addrIn)
	close(cHandler.connIn)
	close(cHandler.peerIn)
}

func (cHandler *connectionHandler) handleAddresses() {

	for addr := range cHandler.addrIn {

		<-cHandler.ticker.C

		conn, err := net.DialTimeout("tcp", addr, time.Second/4)
		if err != nil {
			log.Println("Connection failed:", addr, err)
			continue
		}

		log.Println("Connection established:", addr)

		cHandler.connIn <- conn
	}
}

func (cHandler *connectionHandler) handleConnections() {

	for conn := range cHandler.connIn {

		cHandler.numConns++

		if cHandler.numConns > maxConnsTotal {
			conn.Close()
			continue
		}

		addr := conn.RemoteAddr().String()

		me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
		if err != nil {
			log.Println("Could not parse local address:", addr, err)
			continue
		}

		you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
		if err != nil {
			log.Println("Could not parse remote address:", addr, err)
			continue
		}

		nonce, err := wire.RandomUint64()
		if err != nil {
			log.Println("Could not generate nonce:", addr, err)
			continue
		}

		log.Println("Peer created:", addr)

		peer := NewPeer(conn, me, you, nonce, protocolNetwork, protocolVersion)
		peer.Start(cHandler.peerIn)
		peer.InitHandshake()
	}
}

func (cHandler *connectionHandler) handlePeers() {

	for peer := range cHandler.peerIn {

		if peer.IsConnected() {
			cHandler.peerOut <- peer
		}
	}
}
