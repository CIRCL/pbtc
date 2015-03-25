package all

import (
	"errors"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type peerState uint32

const (
	stateIdle = iota
	statePending
	stateHsFailed
	stateConnected
)

type peer struct {
	conn    net.Conn
	me      *wire.NetAddress
	you     *wire.NetAddress
	nonce   uint64
	network wire.BitcoinNet
	version uint32

	sendEx     chan wire.Message
	recvEx     chan wire.Message
	successOut chan<- *peer
	failOut    chan<- *peer
	addrOut    chan<- string
	state      peerState
	backoff    float64
	suspend    chan struct{}
	outbound   bool
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

	peer := &peer{
		conn:    conn,
		me:      me,
		you:     you,
		nonce:   nonce,
		network: network,
		version: version,
	}

	return peer, nil
}

func (peer *peer) Connection(conn net.Conn) {

	peer.conn = conn
}

func (peer *peer) Disconnect() {

	peer.conn.Close()

	peer.state = stateIdle
}

func (peer *peer) Start() {

	peer.sendEx = make(chan wire.Message, bufferMessage)
	peer.recvEx = make(chan wire.Message, bufferMessage)

	go peer.handleSend()
	go peer.handleRecv()
}

func (peer *peer) Stop() {

	close(peer.sendEx)
	close(peer.recvEx)
}

func (peer *peer) Process(addrOut chan<- string) {

	peer.suspend = make(chan struct{})

	peer.addrOut = addrOut

	go peer.handleMessages()
}

func (peer *peer) Suspend() {

	close(peer.suspend)
}

func (peer *peer) SendMessage(msg wire.Message) {

	peer.sendEx <- msg
}

func (peer *peer) RecvMessage(timeout time.Duration) (wire.Message, error) {

	timer := time.NewTimer(timeout)

	select {

	case msg := <-peer.recvEx:
		return msg, nil

	case <-timer.C:
		return nil, errors.New("Receiving message timed out")
	}
}

func (peer *peer) Handshake(outgoing bool, addrOut chan<- string, peerOut chan<- *peer) {

	if outgoing {
		peer.initHandshake()
	} else {
		peer.waitHandshake()
	}

	switch peer.state {

	case stateHsFailed:
		addrOut <- peer.conn.RemoteAddr().String()

	case stateConnected:
		peerOut <- peer
	}
}

func (peer *peer) initHandshake() {

	peer.state = statePending

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	peer.SendMessage(verOut)

	verIn, err := peer.RecvMessage(timeoutRecv)
	if err != nil {
		log.Println("Outgoing handshake version reply timed out:", err)
		peer.state = stateHsFailed
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:
		// set version
	default:
		log.Println("Outgoing handshake wrong version reply")
		peer.state = stateHsFailed
		return
	}

	verAck := wire.NewMsgVerAck()
	peer.SendMessage(verAck)

	peer.state = stateConnected
}

func (peer *peer) waitHandshake() {

	peer.state = statePending

	verIn, err := peer.RecvMessage(timeoutRecv)
	if err != nil {
		log.Println("Incoming handshake version message timed out:", err)
		peer.state = stateHsFailed
		return
	}

	switch verIn.(type) {
	case *wire.MsgVersion:
		// set version
	default:
		log.Println("Incoming handshake wrong version message")
		peer.state = stateHsFailed
		return
	}

	verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	peer.SendMessage(verOut)

	verAck, err := peer.RecvMessage(timeoutRecv)
	if err != nil {
		log.Println("Incoming handshake verack message timed out:", err)
		peer.state = stateHsFailed
		return
	}

	switch verAck.(type) {
	case *wire.MsgVerAck:
		// nothing
	default:
		log.Println("Incoming handshake wrong verack message")
		peer.state = stateHsFailed
		return
	}

	peer.state = stateConnected
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

	for {

		select {

		case msg := <-peer.recvEx:
			peer.processMessage(msg)

		case <-peer.suspend:
			break
		}
	}
}

func (peer *peer) processMessage(msg wire.Message) {

	for msg := range peer.recvEx {

		switch m := msg.(type) {

		case *wire.MsgVersion:

		case *wire.MsgVerAck:

		case *wire.MsgPing:

		case *wire.MsgPong:

		case *wire.MsgGetAddr:

		case *wire.MsgAddr:
			for _, addr := range m.AddrList {
				peer.addrOut <- net.JoinHostPort(addr.IP.String(), strconv.Itoa(int(addr.Port)))
			}

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
