package all

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type peer struct {
	recvEx chan wire.Message
	sendEx chan wire.Message

	sigSend  chan struct{}
	sigRecv  chan struct{}
	sigRetry chan struct{}
	sigMsgs  chan struct{}

	waitGroup *sync.WaitGroup

	addr      string
	evtOut    chan<- event
	backoff   time.Duration
	state     uint32
	currLevel uint32
	nextLevel uint32
	prevLevel uint32

	conn    net.Conn
	me      *wire.NetAddress
	you     *wire.NetAddress
	nonce   uint64
	network wire.BitcoinNet
	version uint32
}

func NewPeer(addr string, evtOut chan<- event) *peer {
	peer := &peer{
		recvEx: make(chan wire.Message, bufferPeerRecv),
		sendEx: make(chan wire.Message, bufferPeerSend),

		waitGroup: &sync.WaitGroup{},

		addr:    addr,
		evtOut:  evtOut,
		backoff: backoffInitial,
		state:   stateIdle,
	}

	return peer
}

func (peer *peer) Start() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateRunning) {
		return
	}

	log.Println("[PEER]", peer.addr, "Starting")

	handleLevel()

	log.Println("[PEER]", peer.addr, "Started")
}

func (peer *peer) Stop() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateRunning, stateIdle) {
		return
	}

	log.Println("[PEER]", peer.addr, "Stopping")

	close(peer.sigState)

	peer.waitGroup.Wait()

	log.Println("[PEER]", peer.addr, "Stopped")
}

func (peer *peer) handleLevel() {

	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

		for {
			select {
			case _, ok := <-peer.sigState:
				if !ok {
					peer.SetLevel(stateIdle)
					return
				}

			default:
				adjustLevel()
			}
		}
	}()
}

func (peer *peer) adjustLevel() {
	if atomic.LoadUint32(&peer.currLevel) == stateBusy {
		return
	}
}

func (peer *peer) SetLevel(level uint32) {
	atomic.StoreUint32(&peer.nextLevel, level)
}

func (peer *peer) GetLevel() uint32 {
	return atomic.LoadUint32(&peer.currLevel)
}

func (peer *peer) Connect() {
	if !peer.startTransition(stateIdle, statePending) {
		return
	}

	log.Println("[PEER]", peer.addr, "Connecting")

	peer.sigRetry = make(chan struct{}, 1)

	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

		conn, err := net.DialTimeout("tcp", peer.addr, timeoutDial)
		if err != nil {
			peer.abortTransition()
			return
		}

		err = peer.Connection(conn)
		if err != nil {
			conn.Close()
			peer.abortTransition()
			return
		}

		log.Println("[PEER]", peer.addr, "Connected")

		peer.endTransition()
	}()
}

func (peer *peer) Disconnect() {
	if !peer.startTransition(statePending, stateIdle) {
		return
	}

	log.Println("[PEER]", peer.addr, "Disconnecting")

	close(peer.sigRetry)

	peer.conn.Close()

	log.Println("[PEER]", peer.addr, "Disconnected")

	peer.endTransition()
}

func (peer *peer) InitHandshake(network wire.BitcoinNet, version uint32) {
	if !peer.startTransition(statePending, stateReady) {
		return
	}

	log.Println("[PEER]", peer.addr, "Initiating outgoing handshake")

	peer.network = network
	peer.version = version

	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		err := peer.sendMessage(verOut)
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake out verout failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		verIn, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake out verin failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		switch msg := verIn.(type) {
		case *wire.MsgVersion:
			version := uint32(msg.ProtocolVersion)
			if version < peer.version {
				peer.version = version
			}

		default:
			log.Println("[PEER]", peer.addr, "Handshake out verin invalid", msg.Command())
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		verAck := wire.NewMsgVerAck()
		err = peer.sendMessage(verAck)
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake out verack failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		log.Println("[PEER]", peer.addr, "Outgoing handshake complete")

		peer.endTransition()
	}()
}

func (peer *peer) WaitHandshake(network wire.BitcoinNet, version uint32) {
	if !peer.startTransition(statePending, stateReady) {
		return
	}

	log.Println("[PEER]", peer.addr, "Waiting for incoming handshake")

	peer.network = network
	peer.version = version

	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

		verIn, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake in verin failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		switch msg := verIn.(type) {
		case *wire.MsgVersion:
			version := uint32(msg.ProtocolVersion)
			if version < peer.version {
				peer.version = version
			}

		default:
			log.Println("[PEER]", peer.addr, "Handshake in verin invalid", msg.Command())
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		verOut := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
		err = peer.sendMessage(verOut)
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake in verout failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		verAck, err := peer.recvMessage()
		if err != nil {
			log.Println("[PEER]", peer.addr, "Handshake in verack failed", err)
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		switch msg := verAck.(type) {
		case *wire.MsgVerAck:

		default:
			log.Println("[PEER]", peer.addr, "Handshake in verack invalid", msg.Command())
			peer.abortTransition()
			peer.Disconnect()
			return
		}

		log.Println("[PEER]", peer.addr, "Incoming handshake complete")

		peer.endTransition()
	}()
}

func (peer *peer) Suspend() {
	if !peer.startTransition(stateReady, statePending) {
		return
	}

	log.Println("[PEER]", peer.addr, "Suspending")

	log.Println("[PEER]", peer.addr, "Suspended")

	peer.endTransition()
}

func (peer *peer) Start() {
	if !peer.startTransition(stateReady, stateRunning) {
		return
	}

	log.Println("[PEER]", peer.addr, "Starting")

	peer.sigSend = make(chan struct{}, 1)
	peer.sigRecv = make(chan struct{}, 1)
	peer.sigMsgs = make(chan struct{}, 1)

	peer.handleSend()
	peer.handleReceive()
	peer.handleMessages()

	log.Println("[PEER]", peer.addr, "Started")

	peer.endTransition()
}

func (peer *peer) Stop() {
	if !peer.startTransition(stateRunning, stateReady) {
		return
	}

	log.Println("[PEER]", peer.addr, "Stopping")

	close(peer.sigSend)
	close(peer.sigRecv)
	close(peer.sigMsgs)

	log.Println("[PEER]", peer.addr, "Stopped")

	peer.endTransition()
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
	randomFactor := time.Duration(float32(peer.backoff) * backoffRandomizer * rand.Float32())
	backoff := peer.backoff + randomFactor
	timer := time.NewTimer(backoff)
	peer.backoff = time.Duration(float32(peer.backoff) * backoffMultiplier)
	if peer.backoff > backoffMaximum {
		peer.backoff = backoffMaximum
	}

	log.Println("[PEER]", peer.addr, "Retrying", backoff)

	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

		select {
		case _, ok := <-peer.sigRetry:
			if !ok {
				return
			}

		case <-timer.C:
			log.Println("[PEER]", peer.addr, "Queuing for retry")
			peerOut <- peer
			return
		}
	}()
}

func (peer *peer) Close() {
	for atomic.LoadUint32(&peer.state) == stateBusy {
		time.Sleep(time.Millisecond)
	}

	peer.Stop()
	peer.Suspend()
	peer.Disconnect()

	peer.waitGroup.Wait()
}

func (peer *peer) Poll() {
	if atomic.LoadUint32(&peer.state) != stateRunning {
		return
	}

	peer.sendEx <- wire.NewMsgGetAddr()
}

func (peer *peer) Ping() {
	if atomic.LoadUint32(&peer.state) != stateRunning {
		return
	}

	peer.sendEx <- wire.NewMsgPing(peer.nonce)
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
	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

	SendLoop:
		for {
			select {
			case _, ok := <-peer.sigSend:
				if !ok {
					break SendLoop
				}

			case msg := <-peer.sendEx:
				err := peer.sendMessage(msg)
				if e, ok := err.(net.Error); ok && e.Timeout() {
					continue SendLoop
				}
				if err != nil {
					peer.Stop()
					peer.Disconnect()
					break SendLoop
				}
			}
		}
	}()

}

func (peer *peer) handleReceive() {
	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

	RecvLoop:
		for {
			select {
			case _, ok := <-peer.sigRecv:
				if !ok {
					break RecvLoop
				}

			default:
				msg, err := peer.recvMessage()
				if e, ok := err.(net.Error); ok && e.Timeout() {
					continue RecvLoop
				}
				if err != nil {
					peer.Stop()
					peer.Disconnect()
					break RecvLoop
				}

				peer.recvEx <- msg
			}
		}
	}()

}

func (peer *peer) handleMessages() {
	peer.waitGroup.Add(1)

	go func() {
		defer peer.waitGroup.Done()

	MsgLoop:
		for {
			select {
			case _, ok := <-peer.sigMsgs:
				if !ok {
					break MsgLoop
				}

			case msg := <-peer.recvEx:
				peer.processMessage(msg)
			}
		}
	}()

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

func (peer *peer) pushVersion() {
	msg := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	msg.AddUserAgent("Satoshi", "0.9.3")
	msg.Services = wire.SFNodeNetwork
}
