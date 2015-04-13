package all

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/op/go-logging"
)

const (
	stateIdle      = iota // initial state where module is ready to start
	stateConnected        // peer state when it is connected
	stateRunning          // module state after a module was started
	stateBusy             // state used during state changes
	stateShutdown         // irreversible shutdown of module
)

type peer struct {
	mgr      *Manager
	incoming bool
	network  wire.BitcoinNet
	version  uint32
	nonce    uint64
	shaked   bool

	addr  *net.TCPAddr
	conn  *net.TCPConn
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

// newPeer creates a new peer for the given manager, indicating whether the connection
// is incoming or outgoing. It will also set the network and version for communication.
// The peer will be fully initialized, but remain unconnected and in idle state.
func newPeer(mgr *Manager, incoming bool, network wire.BitcoinNet, version uint32,
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

// newIncomingPeer creates a new incoming peer for the given manager and connection.
// It will initialize the peer and then try to parse the necessary information from the connection.
// It does not return the peer as the peer will notify the manager by itself once a
// successful connection was set up.
func newIncomingPeer(mgr *Manager, conn *net.TCPConn, network wire.BitcoinNet, version uint32,
	nonce uint64) error {
	// create the peer with basic required variables
	peer := newPeer(mgr, true, network, version, nonce)

	// here, we try to parse the remote adress as TCP address
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Can only use TCP connections for peers")
	}

	// try to create a net address for the remote address
	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	// try to parse the local address as TCP address
	local, ok := conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Can only use TCP connections for source")
	}

	// try to create a net address for the local address
	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	// set connection paramaters and use the given connection for communication
	peer.addr = addr
	peer.you = you
	peer.me = me
	peer.Use(conn)

	return nil
}

// newOutgoingPeer creates a new outgoing peer for the given manager and address.
// It will initialize the peer and start connection procedures.
// It does not return the peer, as it will notify the manager on its own after successful
// connection.
func newOutgoingPeer(mgr *Manager, addr *net.TCPAddr, network wire.BitcoinNet, version uint32,
	nonce uint64) error {
	// create peer with all basic required information
	peer := newPeer(mgr, false, network, version, nonce)

	// try to create a net address for the given TCP adress
	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	// set remote connection parameters and try connecting to finish up
	peer.addr = addr
	peer.you = you
	go peer.Connect()

	return nil
}

// String returns the remote address of the given peer in string format for printing.
func (peer *peer) String() string {
	return peer.addr.String()
}

// Connect will try to use the configured remote address to set up a connection. If the
// connection fails, we immediately stop the peer as it can't communicate.
func (peer *peer) Connect() {
	// we can only start the peer if it is currently in idle mode
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateBusy) {
		return
	}

	// if we can't establish the connection, abort
	connGen, err := net.DialTimeout("tcp", peer.addr.String(), timeoutDial)
	if err != nil {
		peer.Stop()
		return
	}

	// this should always work
	conn := connGen.(*net.TCPConn)

	// try to parse the local address as tcp address
	local, ok := conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		peer.Stop()
		return
	}

	// try to create a net address for the local address
	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		peer.Stop()
		return
	}

	// set the remaining connection parameters and save the connection
	peer.me = me
	peer.conn = conn

	// if we have shutdown this peer in the mean time, just dump it
	// otherwise, set the state to connected
	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateConnected) {
		peer.Stop()
		return
	}

	peer.mgr.peerNew <- peer
}

// Use will try to use a given connection to communicate with a peer.
func (peer *peer) Use(conn *net.TCPConn) {
	// we can only use a connection if we are still idle
	if !atomic.CompareAndSwapUint32(&peer.state, stateIdle, stateBusy) {
		return
	}

	// if the connection is nil, dump the peer
	if conn == nil {
		peer.Stop()
		return
	}

	// save the given connection
	peer.conn = conn

	// if we have shutdown, simply dump the peer, otherwise, we are now connected
	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateConnected) {
		peer.Stop()
		return
	}

	peer.mgr.peerNew <- peer
}

// Start will try to start the peer. It only works if the peer is already connected.
func (peer *peer) Start() {
	// check if we are in connected state to launch
	if !atomic.CompareAndSwapUint32(&peer.state, stateConnected, stateBusy) {
		return
	}

	// if we are talking to an outgoing peer, we should send the version first
	// if this fails, the handshake broke down and we are done with this peer
	if !peer.incoming {
		peer.pushVersion()
	}

	// add three handlers to our waitgroup and launch them
	// they take care of sending, queuing received messages and processing them
	peer.wg.Add(3)
	go peer.handleSend()
	go peer.handleReceive()
	go peer.handleMessages()

	// if we have shut down in the mean-time, we can dump the peer
	if !atomic.CompareAndSwapUint32(&peer.state, stateBusy, stateRunning) {
		peer.mgr.peerDone <- peer
		return
	}
}

// Stop will cleanly shutdown the peer and wait for handlers to quit. In contrast to
// other modules, a stop on a peer is irreversible.
func (peer *peer) Stop() {
	// if we already called shutdown, we don't need to do it twice
	if atomic.SwapUint32(&peer.state, stateShutdown) == stateShutdown {
		return
	}

	// if we have a connection established, close it
	if peer.conn != nil {
		peer.conn.Close()
	}

	// signal all handlers to stop
	close(peer.sigSend)
	close(peer.sigRecv)
	close(peer.sigMsgs)

	// wait for the handlers to stop
	peer.wg.Wait()

	// we will simply stay in a shutdown state here; no further actions are supposed
	// to be executed on this peer now
}

// sendMessage will attempt to write a message on our connection. It will set the write
// deadline in order to respect the timeout defined in our configuration. It will return
// the error if we didn't succeed.
func (peer *peer) sendMessage(msg wire.Message) error {
	peer.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(peer.conn, msg, peer.version, peer.network)

	return err
}

// recvMessage will attempt to read a message from our connection. It will set the read
// deadline in order to respect the timeout defined in our configuration. It will return
/// the read message as well as the error.
func (peer *peer) recvMessage() (wire.Message, error) {
	peer.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	msg, _, err := wire.ReadMessage(peer.conn, peer.version, peer.network)

	return msg, err
}

// handleSend is the handler responsible for sending messages in the queue. It will
// send messages from the queue and push ping messages if the connection is idling.
func (peer *peer) handleSend() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

	// initialize the idle timer to see when we didn't send for a while
	idleTimer := time.NewTimer(timeoutPing)

SendLoop:
	for {
		select {
		// signal for shutdown, so break outer loop
		case _, ok := <-peer.sigSend:
			if !ok {
				break SendLoop
			}

		// we didnt's send a message in a long time, so send a ping
		case <-idleTimer.C:
			peer.pushPing()

		// try to send the next message in the queue
		// on timeouts, we try the next one, on other errors, we stop the peer
		case msg := <-peer.sendQ:
			err := peer.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue SendLoop
			}
			if err != nil {
				peer.Stop()
				peer.mgr.peerDone <- peer
			}

			// we successfully sent a message, so reset the idle timer
			idleTimer.Reset(timeoutPing)
		}
	}
}

// handleReceive is the handler responsible for receiving messages and pushing them onto
// the reception queue. It does not do any processing so that we read all messages as quickly
// as possible.
func (peer *peer) handleReceive() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

	// initialize the timer to see when we didn't receive in a long time
	idleTimer := time.NewTimer(timeoutIdle)

RecvLoop:
	for {
		select {
		// the peer has shutdown so break outer loop
		case _, ok := <-peer.sigRecv:
			if !ok {
				break RecvLoop
			}

		// we didn't receive a message for too long, so time this peer out and dump
		case <-idleTimer.C:
			peer.Stop()
			peer.mgr.peerDone <- peer

		// each iteration without other action, we try receiving a message for a while
		// if we time out, we try again, on error we quit
		default:
			msg, err := peer.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue RecvLoop
			}
			if err != nil {
				peer.Stop()
				peer.mgr.peerDone <- peer
			}

			// we successfully received a message, so reset the idle timer and push it
			// onte the reception queue for further processing
			idleTimer.Reset(timeoutIdle)
			peer.recvQ <- msg
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (peer *peer) handleMessages() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

MsgsLoop:
	for {
		select {
		// shutdown signal, break outer loop
		case _, ok := <-peer.sigMsgs:
			if !ok {
				break MsgsLoop
			}

		// we read a message from the queue, process it depending on type
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
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received version message", peer)

	if msg.Nonce == peer.nonce {
		log.Warning("%v: detected connection to self, disconnecting", peer)
		peer.Stop()
		peer.mgr.peerDone <- peer
		return
	}

	if peer.shaked {
		log.Notice("%v: received version after handshake", peer)
		return
	}

	if msg.ProtocolVersion < int32(wire.MultipleAddressVersion) {
		log.Notice("%v: detected outdated protocol version", peer)
	}

	peer.version = MinUint32(peer.version, uint32(msg.ProtocolVersion))
	peer.shaked = true

	if peer.incoming {
		peer.pushVersion()
	}

	peer.pushVerAck()
	peer.pushGetAddr()
}

func (peer *peer) handleVerAckMsg(msg *wire.MsgVerAck) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received verack message", peer)
}

func (peer *peer) handlePingMsg(msg *wire.MsgPing) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received ping message", peer)

	if peer.version > wire.BIP0031Version {
		peer.pushPong(msg.Nonce)
	}
}

func (peer *peer) handlePongMsg(msg *wire.MsgPong) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received pong message", peer)
}

func (peer *peer) handleGetAddrMsg(msg *wire.MsgGetAddr) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received getaddr message", peer)
}

func (peer *peer) handleAddrMsg(msg *wire.MsgAddr) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received addr message", peer)
}

func (peer *peer) handleInvMsg(msg *wire.MsgInv) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received inv message", peer)
}

func (peer *peer) handleGetHeadersMsg(msg *wire.MsgGetHeaders) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received getheaders message", peer)
}

func (peer *peer) handleHeadersMsg(msg *wire.MsgHeaders) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received headers message", peer)
}

func (peer *peer) handleGetBlocksMsg(msg *wire.MsgGetBlocks) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received getblocks message", peer)
}

func (peer *peer) handleBlockMsg(msg *wire.MsgBlock) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received block message", peer)
}

func (peer *peer) handleGetDataMsg(msg *wire.MsgGetData) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received getdata message", peer)
}

func (peer *peer) handleTxMsg(msg *wire.MsgTx) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received tx message", peer)
}

func (peer *peer) handleAlertMsg(msg *wire.MsgAlert) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received alert message", peer)
}

func (peer *peer) pushVersion() {
	msg := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	msg.AddUserAgent(userAgentName, userAgentVersion)
	msg.AddrYou.Services = wire.SFNodeNetwork
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(wire.RejectVersion)

	peer.sendQ <- msg
}

func (peer *peer) pushVerAck() {
	msg := wire.NewMsgVerAck()

	peer.sendQ <- msg
}

func (peer *peer) pushGetAddr() {
	msg := wire.NewMsgGetAddr()

	peer.sendQ <- msg
}

func (peer *peer) pushPing() {
	msg := wire.NewMsgPing(peer.nonce)

	peer.sendQ <- msg
}

func (peer *peer) pushPong(nonce uint64) {
	msg := wire.NewMsgPong(nonce)

	peer.sendQ <- msg
}
