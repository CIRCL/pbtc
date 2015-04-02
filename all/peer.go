package all

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	stateIdle = iota
	stateConnected
	stateReady
	stateProcessing
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
	state   uint32

	waitGroup *sync.WaitGroup
}

func NewPeer(addr string, evtOut chan<- event) *peer {
	recvEx := make(chan wire.Message, bufferPeerRecv)
	sendEx := make(chan wire.Message, bufferPeerSend)

	sigStopSend := make(chan struct{}, 1)
	sigStopRecv := make(chan struct{}, 1)
	sigSuspend := make(chan struct{}, 1)

	peer := &peer{
		addr: addr,

		recvEx: recvEx,
		sendEx: sendEx,

		sigStopSend: sigStopSend,
		sigStopRecv: sigStopRecv,
		sigSuspend:  sigSuspend,

		evtOut:  evtOut,
		backoff: backoffInitial,
		state:   stateIdle,

		waitGroup: &sync.WaitGroup{},
	}

	return peer
}

func (peer *peer) GetState() uint32 {
	return peer.state
}

func (peer *peer) Start() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateReady, stateProcessing) {
		return
	}

	peer.waitGroup.Add(3)
	go peer.handleSend()
	go peer.handleReceive()
	go peer.handleMessages()

	peer.notifyState()
}

func (peer *peer) Connect() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateConnected) {
		return
	}

	peer.waitGroup.Add(1)
	go func() {
		conn, err := net.DialTimeout("tcp", peer.addr, timeoutDial)
		if err != nil {
			peer.Stop()
			return
		}

		err = peer.Connection(conn)
		if err != nil {
			peer.Stop()
			return
		}

		peer.notifyState()
		peer.waitGroup.Done()
	}()
}

func (peer *peer) Connection(conn net.Conn) error {
	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		return err
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		return err
	}

	nonce, err := wire.RandomUint64()
	if err != nil {
		return err
	}

	peer.conn = conn
	peer.me = me
	peer.you = you
	peer.nonce = nonce

	return nil
}

func (peer *peer) Retry(peerOut chan<- *peer) {
	randomFactor := time.Duration(float32(peer.backoff) * backoffRandomizer)
	backoff := peer.backoff + randomFactor
	timer := time.NewTimer(backoff)
	peer.backoff = time.Duration(float32(peer.backoff) * backoffMultiplier)
	if peer.backoff > backoffMaximum {
		peer.backoff = backoffMaximum
	}

	peer.waitGroup.Add(1)
	go func() {
		<-timer.C
		peerOut <- peer
		peer.waitGroup.Done()
	}()
}

func (peer *peer) InitHandshake(network wire.BitcoinNet, version uint32) {
	if !atomic.CompareAndSwapUint32(&peer.state, stateConnected, stateReady) {
		return
	}

	peer.network = network
	peer.version = version

	peer.waitGroup.Add(1)
	go func() {
		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		err := peer.sendMessage(verOut)
		if err != nil {
			log.Println("[PEER] Handshake out verout failed", peer.addr, err)
			peer.Stop()
			return
		}

		verIn, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER] Handshake out verin failed", peer.addr, err)
			peer.Stop()
			return
		}

		switch msg := verIn.(type) {
		case *wire.MsgVersion:
			version := uint32(msg.ProtocolVersion)
			if version < peer.version {
				peer.version = version
			}

		default:
			log.Println("[PEER] Handshake out verin invalid", peer.addr, msg.Command())
			peer.Stop()
			return
		}

		verAck := wire.NewMsgVerAck()
		err = peer.sendMessage(verAck)
		if err != nil {
			log.Println("[PEER] Handshake out verack failed", peer.addr, err)
			peer.Stop()
			return
		}

		peer.notifyState()
		peer.waitGroup.Done()
	}()
}

func (peer *peer) WaitHandshake(network wire.BitcoinNet, version uint32) {
	if !atomic.CompareAndSwapUint32(&peer.state, stateConnected, stateReady) {
		return
	}

	peer.network = network
	peer.version = version

	peer.waitGroup.Add(1)
	go func() {
		verIn, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER] Handshake in verin failed", peer.addr, err)
			peer.Stop()
			return
		}

		switch msg := verIn.(type) {
		case *wire.MsgVersion:
			version := uint32(msg.ProtocolVersion)
			if version < peer.version {
				peer.version = version
			}

		default:
			log.Println("[PEER] Handshake in verin invalid", peer.addr, msg.Command())
			peer.Stop()
			return
		}

		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		err = peer.sendMessage(verOut)
		if err != nil {
			log.Println("[PEER] Handshake in verout failed", peer.addr, err)
			peer.Stop()
			return
		}

		verAck, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER] Handshake in verack failed", peer.addr, err)
			peer.Stop()
			return
		}

		switch msg := verAck.(type) {
		case *wire.MsgVerAck:

		default:
			log.Println("[PEER] Handshake in verack invalid", peer.addr, msg.Command())
			peer.Stop()
			return
		}

		peer.notifyState()
		peer.waitGroup.Done()
	}()
}

func (peer *peer) Poll() {
	if atomic.LoadUint32(&peer.state) != stateProcessing {
		return
	}

	peer.sendEx <- wire.NewMsgGetAddr()
}

func (peer *peer) Ping() {
	if atomic.LoadUint32(&peer.state) != stateProcessing {
		return
	}

	peer.sendEx <- wire.NewMsgPing(peer.nonce)
}

func (peer *peer) Stop() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateConnected, stateIdle) &&
		!atomic.CompareAndSwapUint32(&peer.state, stateReady, stateIdle) &&
		!atomic.CompareAndSwapUint32(&peer.state, stateProcessing, stateIdle) {
		return
	}

	peer.sigSuspend <- struct{}{}
	peer.sigStopRecv <- struct{}{}
	peer.sigStopSend <- struct{}{}

	if peer.conn != nil {
		peer.conn.Close()
	}

	peer.waitGroup.Wait()
	peer.notifyState()
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

func (peer *peer) notifyState() {
	peer.evtOut <- NewStateEvent(peer, peer.state)
}

func (peer *peer) handleSend() {
SendLoop:
	for {
		select {
		case <-peer.sigStopSend:
			break SendLoop

		case msg := <-peer.sendEx:
			err := peer.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue SendLoop
			}
			if err != nil {
				log.Println("[PEER] Sending failed", peer.addr, err)
				peer.Stop()
				break SendLoop
			}
		}
	}

	peer.waitGroup.Done()
}

func (peer *peer) handleReceive() {
RecvLoop:
	for {
		select {
		case <-peer.sigStopRecv:
			break RecvLoop

		default:
			msg, err := peer.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue RecvLoop
			}
			if err != nil {
				log.Println("[PEER] Receiving failed", peer.addr, err)
				peer.Stop()
				break RecvLoop
			}

			peer.recvEx <- msg
		}
	}

	peer.waitGroup.Done()
}

func (peer *peer) handleMessages() {
MsgLoop:
	for {
		select {
		case <-peer.sigSuspend:
			break MsgLoop

		case msg := <-peer.recvEx:
			peer.processMessage(msg)
		}
	}

	peer.waitGroup.Done()
}

func (peer *peer) processMessage(msg wire.Message) {
	switch m := msg.(type) {
	case *wire.MsgVersion:

	case *wire.MsgVerAck:

	case *wire.MsgPing:
		peer.sendEx <- wire.NewMsgPong(m.Nonce)

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
