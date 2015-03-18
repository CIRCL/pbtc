package usecases

import (
	"errors"
	"io"
	"log"
	"net"

	"github.com/btcsuite/btcd/wire"
)

type ConnectionRepository interface {
	Store(peer Peer)
	FindByIP(ip net.IP) Peer
}

type Peer struct {
	qSend   chan wire.Message
	qRecv   chan wire.Message
	conn    net.Conn
	Version uint32
	network wire.BitcoinNet
	Me      *wire.NetAddress
	You     *wire.NetAddress
	Inbound bool
}

func NewPeer(conn net.Conn, version uint32, network wire.BitcoinNet, inbound bool) (*Peer, error) {
	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not parse local address")
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not parse remote address")
	}

	qSend := make(chan wire.Message, 128)
	qRecv := make(chan wire.Message, 128)

	peer := &Peer{
		qSend:   qSend,
		qRecv:   qRecv,
		conn:    conn,
		Version: version,
		network: network,
		Me:      me,
		You:     you,
		Inbound: inbound,
	}

	go peer.handleSend()
	go peer.handleRecv()

	return peer, nil
}

func (peer *Peer) Start() {

}

func (peer *Peer) Stop() {
	close(peer.qSend)
	peer.conn.Close()
	close(peer.qRecv)
}

func (peer *Peer) SendMessage(msg wire.Message) {
	peer.qSend <- msg
}

func (peer *Peer) RecvMessage() wire.Message {
	return <-peer.qRecv
}

func (peer *Peer) handleSend() {
	for msg := range peer.qSend {
		err := wire.WriteMessage(peer.conn, msg, peer.Version, peer.network)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (peer *Peer) handleRecv() {
	for {
		msg, _, err := wire.ReadMessage(peer.conn, peer.Version, peer.network)

		if err == io.EOF {
			log.Println("Peer connection closed remotely", peer.You)
			break
		}

		if err != nil {
			log.Println(err)
			break
		}

		peer.qRecv <- msg
	}
}
