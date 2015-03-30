package all

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type peer struct {
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

	msgOut chan<- wire.Message
}

func NewPeer(conn net.Conn) (*peer, error) {

	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		return nil, err
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		return nil, err
	}

	nonce, err := wire.RandomUint64()
	if err != nil {
		return nil, err
	}

	recvEx := make(chan wire.Message)
	sendEx := make(chan wire.Message)

	sigStopSend := make(chan struct{})
	sigStopRecv := make(chan struct{})
	sigSuspend := make(chan struct{})

	peer := &peer{
		conn:  conn,
		me:    me,
		you:   you,
		nonce: nonce,

		recvEx: recvEx,
		sendEx: sendEx,

		sigStopSend: sigStopSend,
		sigStopRecv: sigStopRecv,
		sigSuspend:  sigSuspend,
	}

	return peer, nil
}

func (peer *peer) Start(network wire.BitcoinNet, version uint32) {
	peer.network = network
	peer.version = version

	go peer.handleSend()
	go peer.handleReceive()
}

func (peer *peer) Stop() {
	peer.sigStopRecv <- struct{}{}
	peer.sigStopSend <- struct{}{}
}

func (peer *peer) Process(msgOut chan<- wire.Message) {
	peer.msgOut = msgOut

	go peer.handleMessages()
}

func (peer *peer) Suspend() {
	peer.sigSuspend <- struct{}{}
}

func (peer *peer) InitHandshake(peerOut chan<- *peer) {
	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err := peer.sendMessage(verOut)
	if err != nil {
		return
	}

	verIn, err := peer.recvMessage()
	if err != nil {
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:

	default:
		return
	}

	verAck := wire.NewMsgVerAck()
	err = peer.sendMessage(verAck)
	if err != nil {
		return
	}

	peerOut <- peer
}

func (peer *peer) WaitHandshake(peerOut chan<- *peer) {
	verIn, err := peer.recvMessage()
	if err != nil {
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:

	default:
		return
	}

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err = peer.sendMessage(verOut)
	if err != nil {
		return
	}

	verAck, err := peer.recvMessage()
	if err != nil {
		return
	}

	switch verAck.(type) {
	case *wire.MsgVerAck:

	default:
		return
	}

	peerOut <- peer
}

func (peer *peer) sendMessage(msg wire.Message) error {
	peer.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(peer.conn, msg, peer.version, peer.network)

	return err
}

func (peer *peer) recvMessage() (wire.Message, error) {
	peer.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
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
	switch msg.(type) {
	case *wire.MsgVersion:

	case *wire.MsgVerAck:

	case *wire.MsgPing:

	case *wire.MsgPong:

	case *wire.MsgGetAddr:

	case *wire.MsgAddr:
		peer.msgOut <- msg

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
