package domain

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/usecases"
)

type Initializer struct {
	nodeIn  chan string
	connIn  chan net.Conn
	peerOut chan *usecases.Peer
	version uint32
	network wire.BitcoinNet
	nonce   uint64
}

func NewInitializer(version uint32, network wire.BitcoinNet) (*Initializer, error) {
	nonce, err := wire.RandomUint64()
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not get random nonce for initializer")
	}

	init := &Initializer{
		version: version,
		network: network,
		nonce:   nonce,
	}

	return init, nil
}

func (init *Initializer) Start(nodeIn chan string, connIn chan net.Conn, peerOut chan *usecases.Peer) {
	init.nodeIn = nodeIn
	init.connIn = connIn
	init.peerOut = peerOut

	go init.handleNodes()
	go init.handleConns()
}

func (init *Initializer) Stop() {
	close(init.nodeIn)
	close(init.connIn)
}

func (init *Initializer) AddNode(node string) {
	init.nodeIn <- node
}

func (init *Initializer) AddConn(conn net.Conn) {
	init.connIn <- conn
}

func (init *Initializer) handleNodes() {
	for node := range init.nodeIn {
		conn, err := net.DialTimeout("tcp4", node+":8333", time.Second)
		if err != nil {
			log.Println(err)
			continue
		}

		peer, err := usecases.NewPeer(conn, init.version, init.network, false)
		if err != nil {
			log.Println(err)
			conn.Close()
			continue
		}

		log.Println("Starting handshake for outbound connection", peer.You)
		init.sendVersion(peer)
		go init.waitVersion(peer)
	}
}

func (init *Initializer) handleConns() {
	for conn := range init.connIn {
		peer, err := usecases.NewPeer(conn, init.version, init.network, true)
		if err != nil {
			log.Println(err)
			conn.Close()
			continue
		}

		log.Println("Starting handshake for inbound connection", peer.You)
		go init.waitVersion(peer)
	}
}

func (init *Initializer) sendVersion(peer *usecases.Peer) {
	msg := wire.NewMsgVersion(peer.Me, peer.You, init.nonce, 0)
	peer.SendMessage(msg)
	log.Println("Sent version to", peer.You)
}

func (init *Initializer) sendVerAck(peer *usecases.Peer) {
	msg := wire.NewMsgVerAck()
	peer.SendMessage(msg)
	log.Println("Sent verack to", peer.You)
}

func (init *Initializer) waitVerAck(peer *usecases.Peer) {
	msg := peer.RecvMessage()
	switch msg.(type) {
	case *wire.MsgVerAck:
		log.Println("Received verack from", peer.You)
		if peer.Inbound {
			init.peerOut <- peer
		} else {
			peer.Stop()
		}

	default:
		log.Println("Received wrong message on verack", peer.You)
		peer.Stop()
	}
}

func (init *Initializer) waitVersion(peer *usecases.Peer) {
	msg := peer.RecvMessage()
	switch t := msg.(type) {
	case *wire.MsgVersion:
		log.Println("Received version from", peer.You)
		if peer.Inbound {
			init.setVersion(peer, uint32(t.ProtocolVersion))
			init.sendVersion(peer)
			go init.waitVerAck(peer)
		} else {
			init.setVersion(peer, uint32(t.ProtocolVersion))
			init.sendVerAck(peer)
			init.peerOut <- peer
		}
	default:
		log.Println("Received wrong message on version", peer.You)
		peer.Stop()
	}
}

func (init *Initializer) setVersion(peer *usecases.Peer, version uint32) {
	if version < init.version {
		peer.Version = version
	}
}
