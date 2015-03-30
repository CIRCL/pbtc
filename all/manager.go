package all

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	addrIn chan string
	connIn chan net.Conn
	msgIn  chan wire.Message
	peerIn chan *peer

	signalConnect *time.Ticker

	network wire.BitcoinNet
	version uint32

	peerList map[string]*peer
}

func NewManager() *manager {
	addrIn := make(chan string)
	connIn := make(chan net.Conn)
	msgIn := make(chan wire.Message)
	peerIn := make(chan *peer)

	signalConnect := time.NewTicker(1 * time.Second / maxConnsPerSec)

	peerList := make(map[string]*peer)

	mgr := &manager{
		addrIn: addrIn,
		connIn: connIn,
		msgIn:  msgIn,
		peerIn: peerIn,

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

	go mgr.handleMessages()
	go mgr.handlePeers()
	go mgr.handleConnections()
	go mgr.handleAddresses()
}

func (mgr *manager) Stop() {
	close(mgr.addrIn)
	close(mgr.connIn)
	close(mgr.peerIn)
	close(mgr.msgIn)
}

func (mgr *manager) handleAddresses() {
	for addr := range mgr.addrIn {
		_, ok := mgr.peerList[addr]
		if ok {
			continue
		}

		<-mgr.signalConnect.C

		log.Println("Dialing new connection:", addr)

		conn, err := net.DialTimeout("tcp", addr, timeoutDial)
		if err != nil {
			log.Println("Dialing failed:", addr, err)
			continue
		}

		log.Println("Creating new outgoing peer:", addr)

		peer, err := NewPeer(conn, mgr.network, mgr.version)
		if err != nil {
			log.Println("Creating peer failed:", addr, err)
			conn.Close()
			continue
		}

		log.Println("Peer connected, initiating handshake:", addr)

		go peer.InitHandshake(mgr.peerIn)
		mgr.peerList[addr] = peer
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

		peer, err := NewPeer(conn, mgr.network, mgr.version)
		if err != nil {
			log.Println("Creating peer failed:", addr, err)
			conn.Close()
			continue
		}

		log.Println("Peer connected, waiting for handshake:", addr)

		go peer.WaitHandshake(mgr.peerIn)
		mgr.peerList[addr] = peer
	}
}

func (mgr *manager) handlePeers() {
	for peer := range mgr.peerIn {
		peer.Start(mgr.msgIn)

		log.Println("Handshake complete, initiating message processing")
	}
}

func (mgr *manager) handleMessages() {
	for m := range mgr.msgIn {
		switch msg := m.(type) {
		case *wire.MsgAddr:
			for _, addr := range msg.AddrList {
				mgr.addrIn <- net.JoinHostPort(addr.IP.String(), strconv.Itoa(int(addr.Port)))
			}

		default:
			log.Println("Message received:", msg.Command())

		}
	}
}
