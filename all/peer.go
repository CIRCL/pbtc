package all

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type peer struct {
	mgr      *manager
	incoming bool
	network  wire.BitcoinNet
	version  uint32
	nonce    uint64

	addr  *net.TCPAddr
	conn  net.Conn
	me    *wire.NetAddress
	you   *wire.NetAddress
	state uint32

	wg      *sync.WaitGroup
	sigSend chan struct{}
	sigRecv chan struct{}
	sigMsgs chan struct{}
	sendQ   chan wire.Message
	recvQ   chan wire.Message
}

func newPeer(mgr *manager, incoming bool, network wire.BitcoinNet, version uint32,
	nonce uint64) *peer {
	peer := &peer{
		mgr:      mgr,
		incoming: incoming,
		network:  network,
		version:  version,
		nonce:    nonce,

		wg:      &sync.WaitGroup{},
		sigSend: make(chan struct{}, 1),
		sigRecv: make(chan struct{}, 1),
		sigMsgs: make(chan struct{}, 1),
		sendQ:   make(chan wire.Message, bufferPeerSend),
		recvQ:   make(chan wire.Message, bufferPeerRecv),
	}

	return peer
}

func NewIncomingPeer(mgr *manager, conn net.Conn, network wire.BitcoinNet, version uint32,
	nonce uint64) (*peer, error) {
	peer := newPeer(mgr, true, network, version, nonce)

	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("Can only use TCP connections for peers")
	}

	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return nil, err
	}

	local := conn.LocalAddr()
	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		return nil, err
	}

	peer.addr = addr
	peer.you = you
	peer.me = me
	peer.Use(conn)
	return peer, nil
}

func NewOutgoingPeer(mgr *manager, addr *net.TCPAddr, network wire.BitcoinNet, version uint32,
	nonce uint64) (*peer, error) {
	peer := newPeer(mgr, false, network, version, nonce)

	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return nil, err
	}

	peer.addr = addr
	peer.you = you
	go peer.Connect()
	return peer, nil
}

func (peer *peer) String() string {
	return peer.addr.String()
}

func (peer *peer) Connect() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateBusy) {
		return
	}

	conn, err := net.DialTimeout("tcp", peer.addr.String(), timeoutDial)
	if err != nil {
		peer.Stop()
		return
	}

	local := conn.LocalAddr()
	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		peer.Stop()
		return
	}

	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateConnected) {
		return
	}

	peer.me = me
	peer.conn = conn
	peer.mgr.peerNew <- peer
}

func (peer *peer) Use(conn net.Conn) {
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateBusy) {
		return
	}

	if conn == nil {
		peer.Stop()
		return
	}

	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateConnected) {
		return
	}

	peer.conn = conn
	peer.mgr.peerNew <- peer
}

func (peer *peer) Start() {
	if !atomic.CompareAndSwapUint32(&peer.state, stateConnected, stateBusy) {
		return
	}

	if !peer.incoming {
		err := peer.pushVersion()
		if err != nil {
			peer.Stop()
		}
	}

	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateRunning) {
		return
	}

	peer.wg.Add(3)
	go peer.handleSend()
	go peer.handleReceive()
	go peer.handleMessages()
}

func (peer *peer) Stop() {
	if atomic.SwapUint32(&peer.state, stateShutdown) == stateShutdown {
		return
	}

	if peer.conn != nil {
		peer.conn.Close()
	}

	close(peer.sigSend)
	close(peer.sigRecv)
	close(peer.sigMsgs)
	peer.wg.Wait()
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
	defer peer.wg.Done()

	idleTimer := time.NewTimer(timeoutPing)

SendLoop:
	for {
		select {
		case _, ok := <-peer.sigSend:
			if !ok {
				break SendLoop
			}

		case <-idleTimer.C:
			err := peer.pushPing()
			if err != nil {
				peer.Stop()
			}

		case msg := <-peer.sendQ:
			err := peer.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue SendLoop
			}
			if err != nil {
				peer.Stop()
			}

			idleTimer.Reset(timeoutPing)
		}
	}
}

func (peer *peer) handleReceive() {
	defer peer.wg.Done()

	idleTimer := time.NewTimer(timeoutIdle)

RecvLoop:
	for {
		select {
		case _, ok := <-peer.sigRecv:
			if !ok {
				break RecvLoop
			}

		case <-idleTimer.C:
			peer.Stop()

		default:
			msg, err := peer.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue RecvLoop
			}
			if err != nil {
				peer.Stop()
			}

			idleTimer.Reset(timeoutIdle)
			peer.recvQ <- msg
		}
	}
}

func (peer *peer) handleMessages() {
	defer peer.wg.Done()

MsgsLoop:
	for {
		select {
		case _, ok := <-peer.sigMsgs:
			if !ok {
				break MsgsLoop
			}

		case msg := <-peer.recvQ:
			switch m := msg.(type) {
			case *wire.MsgVersion:
				peer.handleVersionMsg(m)

			case *wire.MsgVerAck:
				peer.handleVerAckMsg(m)

			case *wire.MsgPing:
				peer.handlePingMsg(m)

			case *wire.MsgPong:
				peer.handlePongMsg(m)

			case *wire.MsgGetAddr:
				peer.handleGetAddrMsg(m)

			case *wire.MsgAddr:
				peer.handleAddrMsg(m)

			case *wire.MsgInv:
				peer.handleInvMsg(m)

			case *wire.MsgGetHeaders:
				peer.handleGetHeadersMsg(m)

			case *wire.MsgHeaders:
				peer.handleHeadersMsg(m)

			case *wire.MsgGetBlocks:
				peer.handleGetBlocksMsg(m)

			case *wire.MsgBlock:
				peer.handleBlockMsg(m)

			case *wire.MsgGetData:
				peer.handleGetDataMsg(m)

			case *wire.MsgTx:
				peer.handleTxMsg(m)

			case *wire.MsgAlert:
				peer.handleAlertMsg(m)

			default:
			}
		}
	}
}

func (peer *peer) handleVersionMsg(msg *wire.MsgVersion) {
}

func (peer *peer) handleVerAckMsg(msg *wire.MsgVerAck) {

}

func (peer *peer) handlePingMsg(msg *wire.MsgPing) {

}

func (peer *peer) handlePongMsg(msg *wire.MsgPong) {

}

func (peer *peer) handleGetAddrMsg(msg *wire.MsgGetAddr) {

}

func (peer *peer) handleAddrMsg(msg *wire.MsgAddr) {

}

func (peer *peer) handleInvMsg(msg *wire.MsgInv) {

}

func (peer *peer) handleGetHeadersMsg(msg *wire.MsgGetHeaders) {

}

func (peer *peer) handleHeadersMsg(msg *wire.MsgHeaders) {

}

func (peer *peer) handleGetBlocksMsg(msg *wire.MsgGetBlocks) {

}

func (peer *peer) handleBlockMsg(msg *wire.MsgBlock) {

}

func (peer *peer) handleGetDataMsg(msg *wire.MsgGetData) {

}

func (peer *peer) handleTxMsg(msg *wire.MsgTx) {

}

func (peer *peer) handleAlertMsg(msg *wire.MsgAlert) {

}

func (peer *peer) pushVersion() error {
	msg := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	msg.AddUserAgent(userAgentName, userAgentVersion)
	msg.AddrYou.Services = wire.SFNodeNetwork
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(wire.RejectVersion)

	return peer.sendMessage(msg)
}

func (peer *peer) pushGetaddr() error {
	msg := wire.NewMsgGetAddr()

	return peer.sendMessage(msg)
}

func (peer *peer) pushPing() error {
	msg := wire.NewMsgPing(peer.nonce)

	return peer.sendMessage(msg)
}
