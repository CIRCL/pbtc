package peer

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/log"
	"github.com/CIRCL/pbtc/util"
)

const (
	bufferSend   = 1
	bufferRecv   = 1
	timeoutDial  = 1 * time.Second
	timeoutSend  = 1 * time.Second
	timeoutRecv  = 1 * time.Second
	timeoutPing  = 1 * time.Minute
	timeoutIdle  = 3 * time.Minute
	agentName    = "satoshi"
	agentVersion = "0.7.3"
)

type Peer struct {
	wg      *sync.WaitGroup
	sigSend chan struct{}
	sigRecv chan struct{}
	sigMsgs chan struct{}
	sendQ   chan wire.Message
	recvQ   chan wire.Message

	mgr Manager
	log log.Logger

	network wire.BitcoinNet
	version uint32
	nonce   uint64
	addr    *net.TCPAddr
	conn    *net.TCPConn
	me      *wire.NetAddress
	you     *wire.NetAddress

	done uint32
}

func New(options ...func(*Peer)) (*Peer, error) {
	peer := &Peer{
		wg:      &sync.WaitGroup{},
		sigSend: make(chan struct{}, 1),
		sigRecv: make(chan struct{}, 1),
		sigMsgs: make(chan struct{}, 1),
		sendQ:   make(chan wire.Message, bufferSend),
		recvQ:   make(chan wire.Message, bufferRecv),
	}

	for _, option := range options {
		option(peer)
	}

	if peer.addr == nil && peer.conn == nil {
		return nil, errors.New("Must provide address or connection")
	}

	if peer.mgr == nil {
		peer.mgr = &ManagerStub{}
	}

	if peer.log == nil {
		peer.log = &log.LoggerStub{}
	}

	if peer.network == 0 {
		peer.network = wire.TestNet3
	}

	if peer.version == 0 {
		peer.version = wire.RejectVersion
	}

	if peer.nonce == 0 {
		peer.nonce, _ = wire.RandomUint64()
	}

	if peer.conn == nil {
		peer.connect()
		return peer, nil
	}

	err := peer.parse()
	if err != nil {
		return nil, err
	}

	peer.start()

	return peer, nil
}

func SetManager(mgr Manager) func(*Peer) {
	return func(peer *Peer) {
		peer.mgr = mgr
	}
}

func SetLogger(log log.Logger) func(*Peer) {
	return func(peer *Peer) {
		peer.log = log
	}
}

func SetNetwork(network wire.BitcoinNet) func(*Peer) {
	return func(peer *Peer) {
		peer.network = network
	}
}

func SetVersion(version uint32) func(*Peer) {
	return func(peer *Peer) {
		peer.version = version
	}
}

func SetNonce(nonce uint64) func(*Peer) {
	return func(peer *Peer) {
		peer.nonce = nonce
	}
}

func SetAddress(addr *net.TCPAddr) func(*Peer) {
	return func(peer *Peer) {
		peer.addr = addr
	}
}

func SetConnection(conn *net.TCPConn) func(*Peer) {
	return func(peer *Peer) {
		peer.conn = conn
	}
}

func (peer *Peer) String() string {
	return peer.addr.String()
}

func (peer *Peer) Addr() *net.TCPAddr {
	return peer.addr
}

func (peer *Peer) Stop() {
	peer.shutdown()
	peer.wg.Wait()
}

func (peer *Peer) connect() {
	// if we can't establish the connection, abort
	connGen, err := net.DialTimeout("tcp", peer.addr.String(), timeoutDial)
	if err != nil {
		peer.shutdown()
		return
	}

	// this should always work
	conn, ok := connGen.(*net.TCPConn)
	if !ok {
		peer.shutdown()
		return
	}

	peer.conn = conn

	// this should also always work...
	err = peer.parse()
	if err != nil {
		peer.shutdown()
		return
	}

	peer.start()
}

func (peer *Peer) parse() error {
	addr, ok := peer.conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Could not parse remote address from connection")
	}

	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	local, ok := peer.conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Could not parse local address from connection")
	}

	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	peer.addr = addr
	peer.you = you
	peer.me = me

	return nil
}

func (peer *Peer) start() {
	peer.wg.Add(3)
	go peer.goSend()
	go peer.goReceive()
	go peer.goMessages()
}

func (peer *Peer) shutdown() {
	// if we already called shutdown, we don't need to do it twice
	if atomic.SwapUint32(&peer.done, 1) == 1 {
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

	peer.mgr.Stopped(peer)
}

// sendMessage will attempt to write a message on our connection. It will set the write
// deadline in order to respect the timeout defined in our configuration. It will return
// the error if we didn't succeed.
func (peer *Peer) sendMessage(msg wire.Message) error {
	peer.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(peer.conn, msg, peer.version, peer.network)

	return err
}

// recvMessage will attempt to read a message from our connection. It will set the read
// deadline in order to respect the timeout defined in our configuration. It will return
/// the read message as well as the error.
func (peer *Peer) recvMessage() (wire.Message, error) {
	peer.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	msg, _, err := wire.ReadMessage(peer.conn, peer.version, peer.network)

	return msg, err
}

// handleSend is the handler responsible for sending messages in the queue. It will
// send messages from the queue and push ping messages if the connection is idling.
func (peer *Peer) goSend() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

	// initialize the idle timer to see when we didn't send for a while
	idleTimer := time.NewTimer(timeoutPing)

	for atomic.LoadUint32(&peer.done) == 0 {
		select {
		// signal for shutdown, so break outer loop
		case _, ok := <-peer.sigSend:
			if !ok {
				break
			}

		// we didnt's send a message in a long time, so send a ping
		case <-idleTimer.C:
			peer.sendQ <- peer.createPingMsg()

		// try to send the next message in the queue
		// on timeouts, we skip to next one, on other errors, we stop the peer
		case msg := <-peer.sendQ:
			err := peer.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil {
				peer.shutdown()
			}

			// we successfully sent a message, so reset the idle timer
			idleTimer.Reset(timeoutPing)
		}
	}
}

// handleReceive is the handler responsible for receiving messages and pushing them onto
// the reception queue. It does not do any processing so that we read all messages as quickly
// as possible.
func (peer *Peer) goReceive() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

	// initialize the timer to see when we didn't receive in a long time
	idleTimer := time.NewTimer(timeoutIdle)

	for atomic.LoadUint32(&peer.done) == 0 {
		select {
		// the peer has shutdown so break outer loop
		case _, ok := <-peer.sigRecv:
			if !ok {
				break
			}

		// we didn't receive a message for too long, so time this peer out and dump
		case <-idleTimer.C:
			peer.shutdown()

		// each iteration without other action, we try receiving a message for a while
		// if we time out, we try again, on error we quit
		default:
			msg, err := peer.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil {
				peer.shutdown()
			}

			// we successfully received a message, so reset the idle timer and push it
			// onte the reception queue for further processing
			idleTimer.Reset(timeoutIdle)
			peer.recvQ <- msg
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (peer *Peer) goMessages() {
	// let the waitgroup know when we are done
	defer peer.wg.Done()

	for atomic.LoadUint32(&peer.done) == 0 {
		select {
		// shutdown signal, break outer loop
		case _, ok := <-peer.sigMsgs:
			if !ok {
				break
			}

		// we read a message from the queue, process it depending on type
		case msg := <-peer.recvQ:
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

			}
		}
	}
}

func (peer *Peer) handleVersionMsg(msg *wire.MsgVersion) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: received version message", peer)

	if msg.Nonce == peer.nonce {
		log.Warning("%v: detected connection to self, disconnecting", peer)
		peer.shutdown()
		return
	}

	/*if peer.shaked {
		log.Notice("%v: received version after handshake", peer)
		return
	}*/

	if msg.ProtocolVersion < int32(wire.MultipleAddressVersion) {
		log.Notice("%v: detected outdated protocol version", peer)
	}

	peer.version = util.MinUint32(peer.version, uint32(msg.ProtocolVersion))
	//peer.shaked = true

	/*if peer.incoming {
		peer.pushVersion()
	}

	peer.pushVerAck()*/
}

func (peer *Peer) createVersionMsg() *wire.MsgVersion {
	msg := wire.NewMsgVersion(peer.me, peer.you, peer.nonce, 0)
	msg.AddUserAgent(agentName, agentVersion)
	msg.AddrYou.Services = wire.SFNodeNetwork
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(wire.RejectVersion)

	return msg
}

func (peer *Peer) createVerAckMsg() *wire.MsgVerAck {
	msg := wire.NewMsgVerAck()

	return msg
}

func (peer *Peer) createGetAddrMsg() *wire.MsgGetAddr {
	msg := wire.NewMsgGetAddr()

	return msg
}

func (peer *Peer) createPingMsg() *wire.MsgPing {
	msg := wire.NewMsgPing(peer.nonce)

	return msg
}

func (peer *Peer) createPongMsg(nonce uint64) *wire.MsgPong {
	msg := wire.NewMsgPong(nonce)

	return msg
}
