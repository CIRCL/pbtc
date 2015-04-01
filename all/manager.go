package all

import (
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	addrIn chan string
	connIn chan net.Conn
	evtIn  chan event

	signalConnect *time.Ticker

	network wire.BitcoinNet
	version uint32

	peerList map[string]*peer
}

func NewManager() *manager {
	addrIn := make(chan string)
	connIn := make(chan net.Conn)
	evtIn := make(chan event)

	signalConnect := time.NewTicker(1 * time.Second / maxConnsPerSec)
	peerList := make(map[string]*peer)

	mgr := &manager{
		addrIn: addrIn,
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
}

func (mgr *manager) handleAddresses() {
	for addr := range mgr.addrIn {
		_, ok := mgr.peerList[addr]
		if ok {
			continue
		}

		peer := NewPeer(addr, mgr.evtIn)
		mgr.peerList[addr] = peer

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
				// retry
			} else {
				go e.peer.InitHandshake(mgr.network, mgr.version)
			}

		case *eventHandshake:
			if e.err != nil {
				e.peer.Disconnect()
				// retry
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
