package all

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type peer struct {
	addr string

	conn    net.Conn
	me      *wire.NetAddress
	you     *wire.NetAddress
	nonce   uint64
	network wire.BitcoinNet
	version uint32

	recvEx chan wire.Message
	sendEx chan wire.Message

	sigStopSend chan struct{}
	sigStopRecv chan struct{}
	sigSuspend  chan struct{}

	evtOut  chan<- event
	backoff time.Duration
}

func NewPeer(addr string, evtOut chan<- event) *peer {
	recvEx := make(chan wire.Message)
	sendEx := make(chan wire.Message)

	sigStopSend := make(chan struct{})
	sigStopRecv := make(chan struct{})
	sigSuspend := make(chan struct{})

	peer := &peer{
		addr: addr,

		recvEx: recvEx,
		sendEx: sendEx,

		sigStopSend: sigStopSend,
		sigStopRecv: sigStopRecv,
		sigSuspend:  sigSuspend,

		evtOut:  evtOut,
		backoff: backoffInitial,
	}

	return peer
}

func (peer *peer) Start() {
	go peer.handleSend()
	go peer.handleReceive()
	go peer.handleMessages()

	peer.sendEx <- wire.NewMsgGetAddr()
}

func (peer *peer) Stop() {
	peer.sigSuspend <- struct{}{}
	peer.sigStopRecv <- struct{}{}
	peer.sigStopSend <- struct{}{}
}

func (peer *peer) Connect() {
	conn, err := net.DialTimeout("tcp", peer.addr, timeoutDial)
	if err != nil {
		peer.evtOut <- NewConnectionEvent(peer, err)
		return
	}

	peer.Connection(conn)
}

func (peer *peer) Retry(peerOut chan<- *peer) {
	randomFactor := time.Duration(float32(peer.backoff) * backoffRandomizer)
	backoff := peer.backoff + randomFactor
	timer := time.NewTimer(backoff)
	peer.backoff = time.Duration(float32(peer.backoff) * backoffMultiplier)
	if peer.backoff > backoffMaximum {
		peer.backoff = backoffMaximum
	}

	<-timer.C
	peerOut <- peer
}

func (peer *peer) Disconnect() {
	peer.conn.Close()
}

func (peer *peer) Connection(conn net.Conn) {
	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		peer.evtOut <- NewConnectionEvent(peer, err)
		return
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		peer.evtOut <- NewConnectionEvent(peer, err)
		return
	}

	nonce, err := wire.RandomUint64()
	if err != nil {
		peer.evtOut <- NewConnectionEvent(peer, err)
		return
	}

	peer.conn = conn
	peer.me = me
	peer.you = you
	peer.nonce = nonce

	peer.evtOut <- NewConnectionEvent(peer, nil)
}

func (peer *peer) InitHandshake(network wire.BitcoinNet, version uint32) {
	peer.network = network
	peer.version = version

	log.Println("Sending outgoing version message:", peer.conn.RemoteAddr().String())

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err := peer.sendMessage(verOut)
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	log.Println("Waiting for version message reply:", peer.conn.RemoteAddr().String())

	verIn, err := peer.recvMessage()
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:

	default:
		peer.evtOut <- NewHandshakeEvent(peer, errors.New("Wrong handshake reply message type"))
		return
	}

	log.Println("Sending outgoing verack message:", peer.conn.RemoteAddr().String())

	verAck := wire.NewMsgVerAck()
	err = peer.sendMessage(verAck)
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	log.Println("Outgoing handshake complete:", peer.conn.RemoteAddr().String())

	peer.evtOut <- NewHandshakeEvent(peer, nil)
}

func (peer *peer) WaitHandshake(network wire.BitcoinNet, version uint32) {
	peer.network = network
	peer.version = version

	log.Println("Waiting for incoming version message:", peer.conn.RemoteAddr().String())

	verIn, err := peer.recvMessage()
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:

	default:
		peer.evtOut <- NewHandshakeEvent(peer, errors.New("Wrong handshake hello message type"))
		return
	}

	log.Println("Sending version message reply:", peer.conn.RemoteAddr().String())

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err = peer.sendMessage(verOut)
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	log.Println("Waiting for incoming verack message:", peer.conn.RemoteAddr().String())

	verAck, err := peer.recvMessage()
	if err != nil {
		peer.evtOut <- NewHandshakeEvent(peer, err)
		return
	}

	switch verAck.(type) {
	case *wire.MsgVerAck:

	default:
		peer.evtOut <- NewHandshakeEvent(peer, errors.New("Wrong handshake ack message type"))
		return
	}

	log.Println("Incoming handshake complete:", peer.conn.RemoteAddr().String())

	peer.evtOut <- NewHandshakeEvent(peer, nil)
}

func (peer *peer) sendMessage(msg wire.Message) error {
	peer.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(peer.conn, msg, peer.version, peer.network)

	return err
}

func (peer *peer) recvMessage() (wire.Message, error) {
	//peer.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	msg, _, err := wire.ReadMessage(peer.conn, peer.version, peer.network)

	return msg, err
}

func (peer *peer) handleSend() {
	for {
		select {
		case <-peer.sigStopSend:
			break

		case msg := <-peer.sendEx:
			err := peer.sendMessage(msg)
			if err != nil {
				continue
			}
		}
	}
}

func (peer *peer) handleReceive() {
	for {
		select {
		case <-peer.sigStopRecv:
			break

		default:
			peer.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
			msg, err := peer.recvMessage()
			if err != nil {
				continue
			}

			peer.recvEx <- msg
		}
	}
}

func (peer *peer) handleMessages() {
	for {
		select {
		case <-peer.sigSuspend:
			break

		case msg := <-peer.recvEx:
			peer.processMessage(msg)
		}
	}
}

func (peer *peer) processMessage(msg wire.Message) {
	switch m := msg.(type) {
	case *wire.MsgVersion:

	case *wire.MsgVerAck:

	case *wire.MsgPing:

	case *wire.MsgPong:

	case *wire.MsgGetAddr:

	case *wire.MsgAddr:
		peer.evtOut <- NewAddressEvent(peer.addr, m.AddrList)

	case *wire.MsgInv:

	case *wire.MsgGetHeaders:

	case *wire.MsgHeaders:

	case *wire.MsgGetBlocks:

	case *wire.MsgBlock:

	case *wire.MsgGetData:

	case *wire.MsgTx:

	case *wire.MsgAlert:

	default:

	}
}
