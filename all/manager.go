package all

import (
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	addrIn chan string
	peerIn chan *peer
	connIn chan net.Conn
	evtIn  chan event

	signalConnect *time.Ticker

	network wire.BitcoinNet
	version uint32

	peerList map[string]*peer
}

func NewManager() *manager {
	addrIn := make(chan string, bufferManagerAddress)
	peerIn := make(chan *peer, bufferManagerPeer)
	connIn := make(chan net.Conn, bufferManagerConnection)
	evtIn := make(chan event, bufferManagerEvent)

	signalConnect := time.NewTicker(1 * time.Second / maxConnsPerSec)
	peerList := make(map[string]*peer)

	mgr := &manager{
		addrIn: addrIn,
		peerIn: peerIn,
		connIn: connIn,
		evtIn:  evtIn,

		signalConnect: signalConnect,

		peerList: peerList,
	}

	return mgr
}

func (mgr *manager) GetAddrIn() chan<- string {
	return mgr.addrIn
}

func (mgr *manager) GetConnIn() chan<- net.Conn {
	return mgr.connIn
}

func (mgr *manager) Start(network wire.BitcoinNet, version uint32) {
	mgr.network = network
	mgr.version = version

	go mgr.handleAddresses()
	go mgr.handleConnections()
	go mgr.handleEvents()
}

func (mgr *manager) Stop() {
	close(mgr.addrIn)
	close(mgr.connIn)
	close(mgr.evtIn)
}

func (mgr *manager) handleAddresses() {
	for addr := range mgr.addrIn {
		_, ok := mgr.peerList[addr]
		if ok {
			continue
		}

		peer := NewPeer(addr, mgr.evtIn)
		mgr.peerList[addr] = peer

		mgr.peerIn <- peer
	}
}

func (mgr *manager) handlePeers() {
	for peer := range mgr.peerIn {
		<-mgr.signalConnect.C

		go peer.Connect()
	}
}

func (mgr *manager) handleConnections() {
	for conn := range mgr.connIn {
		addr := conn.RemoteAddr().String()
		_, ok := mgr.peerList[addr]
		if ok {
			conn.Close()
			continue
		}

		log.Println("Creating new incoming peer:", addr)

		peer := NewPeer(addr, mgr.evtIn)
		mgr.peerList[addr] = peer

		peer.Connection(conn)
		go peer.WaitHandshake(mgr.network, mgr.version)
	}
}

func (mgr *manager) handleEvents() {
	for event := range mgr.evtIn {
		switch e := event.(type) {
		case *eventConnection:
			if e.err != nil {
				go e.peer.Retry(mgr.peerIn)
			} else {
				go e.peer.InitHandshake(mgr.network, mgr.version)
			}

		case *eventHandshake:
			if e.err != nil {
				e.peer.Disconnect()
				go e.peer.Retry(mgr.peerIn)
			} else {
				go e.peer.Start()
			}

		case *eventAddress:
			for _, addr := range e.list {
				mgr.addrIn <- addr
			}
		}
	}
}
