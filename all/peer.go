package all

import (
	"log"
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

func NewPeer(conn net.Conn, network wire.BitcoinNet, version uint32) (*peer, error) {

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

func (peer *peer) Start(msgOut chan<- wire.Message) {
	peer.msgOut = msgOut

	peer.sendEx <- wire.NewGetAddr()

	go peer.handleMessages()
	go peer.handleSend()
	go peer.handleReceive()
}

func (peer *peer) Stop() {
	peer.sigSuspend <- struct{}{}
	peer.sigStopRecv <- struct{}{}
	peer.sigStopSend <- struct{}{}
}

func (peer *peer) InitHandshake(peerOut chan<- *peer) {

	log.Println("Sending outgoing version message:", peer.conn.RemoteAddr().String())

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err := peer.sendMessage(verOut)
	if err != nil {
		log.Println("Could not send outgoing version message:", peer.conn.RemoteAddr().String(), err)
		return
	}

	log.Println("Waiting for version message reply:", peer.conn.RemoteAddr().String())

	verIn, err := peer.recvMessage()
	if err != nil {
		log.Println("Did not receive version message reply:", peer.conn.RemoteAddr().String(), err)
		return
	}

	switch msg := verIn.(type) {
	case *wire.MsgVersion:

	default:
		log.Println("Version message reply wrong type:", peer.conn.RemoteAddr().String(), msg.Command())
		return
	}

	log.Println("Sending outgoing verack message:", peer.conn.RemoteAddr().String())

	verAck := wire.NewMsgVerAck()
	err = peer.sendMessage(verAck)
	if err != nil {
		log.Println("Could not send outgoing verack message:", peer.conn.RemoteAddr().String(), err)
		return
	}

	log.Println("Outgoing handshake complete:", peer.conn.RemoteAddr().String())
	peerOut <- peer
}

func (peer *peer) WaitHandshake(peerOut chan<- *peer) {

	log.Println("Waiting for incoming version message:", peer.conn.RemoteAddr().String())

	verIn, err := peer.recvMessage()
	if err != nil {
		log.Println("Did not receive incoming version message:", peer.conn.RemoteAddr().String(), err)
		return
	}

	switch msg := verIn.(type) {
	case *wire.MsgVersion:

	default:
		log.Println("Incoming version message wrong type:", peer.conn.RemoteAddr().String(), msg.Command())
		return
	}

	log.Println("Sending version message reply:", peer.conn.RemoteAddr().String())

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	err = peer.sendMessage(verOut)
	if err != nil {
		log.Println("Could not send version message reply:", peer.conn.RemoteAddr().String(), err)
		return
	}

	log.Println("Waiting for incoming verack message:", peer.conn.RemoteAddr().String())

	verAck, err := peer.recvMessage()
	if err != nil {
		log.Println("Did not receive incoming verack message:", peer.conn.RemoteAddr().String(), err)
		return
	}

	switch msg := verAck.(type) {
	case *wire.MsgVerAck:

	default:
		log.Println("Incoming verack message wrong type:", peer.conn.RemoteAddr().String(), msg.Command())
		return
	}

	log.Println("Incoming handshake complete:", peer.conn.RemoteAddr().String())
	peerOut <- peer
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
