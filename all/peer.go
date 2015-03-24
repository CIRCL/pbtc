package all

import (
	"log"
	"net"

	"github.com/btcsuite/btcd/wire"
)

type peerState uint32

const (
	stateIdle = iota
	stateConnected
)

type peer struct {
	sendEx  chan wire.Message
	recvEx  chan wire.Message
	peerOut chan<- *peer
	msgOut  chan<- wire.Message
	conn    net.Conn
	me      *wire.NetAddress
	you     *wire.NetAddress
	nonce   uint64
	network wire.BitcoinNet
	version uint32
	state   peerState
}

func NewPeer(conn net.Conn, me *wire.NetAddress, you *wire.NetAddress, nonce uint64,
	network wire.BitcoinNet, version uint32) *peer {

	peer := &peer{
		conn:    conn,
		me:      me,
		you:     you,
		nonce:   nonce,
		network: network,
		version: version,
		state:   stateIdle,
	}

	return peer
}

func (peer *peer) GetAddress() string {

	return peer.conn.RemoteAddr().String()
}

func (peer *peer) InitHandshake() {

	go func() {

		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		peer.SendMessage(verOut)

		verIn := peer.RecvMessage()
		switch verIn.(type) {
		case *wire.MsgVersion:
			// set version
		default:
			log.Println("Outgoing handshake failed, expected version message")
			break
		}

		verAck := wire.NewMsgVerAck()
		peer.SendMessage(verAck)
		peer.state = stateConnected

		log.Println("Handshake complete:", peer.GetAddress())

		peer.peerOut <- peer
	}()
}

func (peer *peer) WaitHandshake() {

	go func() {

		verIn := peer.RecvMessage()
		switch verIn.(type) {
		case *wire.MsgVersion:
			// set version
		default:
			log.Println("Incoming handshake failed, expected version message")
			break
		}

		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		peer.SendMessage(verOut)

		verAck := peer.RecvMessage()
		switch verAck.(type) {
		case *wire.MsgVerAck:
			// nothing
		default:
			log.Println("Incoming handshake failed, expected verack message")
			break
		}

		log.Println("Handshake complete:", peer.GetAddress())

		peer.state = stateConnected
		peer.peerOut <- peer
	}()
}

func (peer *peer) IsConnected() bool {

	return peer.state == stateConnected
}

func (peer *peer) SendMessage(msg wire.Message) {

	peer.sendEx <- msg
}

func (peer *peer) RecvMessage() wire.Message {

	return <-peer.recvEx
}

func (peer *peer) Start(peerOut chan<- *peer) {

	peer.peerOut = peerOut
	peer.sendEx = make(chan wire.Message, bufferMessage)
	peer.recvEx = make(chan wire.Message, bufferMessage)

	go peer.handleSend()
	go peer.handleRecv()
}

func (peer *peer) Process(msgOut chan<- wire.Message) {

	peer.msgOut = msgOut

	go peer.handleMessages()
}

func (peer *peer) handleSend() {

	for msg := range peer.sendEx {

		err := wire.WriteMessage(peer.conn, msg, peer.version, peer.network)
		if err != nil {
			break
		}
	}
}

func (peer *peer) handleRecv() {

	for {
		msg, _, err := wire.ReadMessage(peer.conn, peer.version, peer.network)
		if err != nil {
			break
		}

		peer.recvEx <- msg
	}
}

func (peer *peer) handleMessages() {

	for msg := range peer.recvEx {

		switch msg.(type) {

		case *wire.MsgVersion:

		case *wire.MsgVerAck:

		case *wire.MsgPing:

		case *wire.MsgPong:

		case *wire.MsgGetAddr:

		case *wire.MsgAddr:

		case *wire.MsgInv:

		case *wire.MsgGetHeaders:

		case *wire.MsgHeaders:

		case *wire.MsgGetBlocks:

		case *wire.MsgBlock:

		case *wire.MsgGetData:

		case *wire.MsgTx:

		case *wire.MsgAlert:

		default:
			log.Println("Unhandled message type:", msg.Command())
		}
	}
}
