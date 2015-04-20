package peer

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/logger"
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
	log logger.Logger

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
	p := &Peer{
		wg:      &sync.WaitGroup{},
		sigSend: make(chan struct{}, 1),
		sigRecv: make(chan struct{}, 1),
		sigMsgs: make(chan struct{}, 1),
		sendQ:   make(chan wire.Message, bufferSend),
		recvQ:   make(chan wire.Message, bufferRecv),

		network: wire.TestNet3,
		version: wire.RejectVersion,
		nonce:   0,
	}

	for _, option := range options {
		option(p)
	}

	if p.addr == nil && p.conn == nil {
		return nil, errors.New("Must provide address or connection")
	}

	if p.conn == nil {
		p.connect()
		return p, nil
	}

	err := p.parse()
	if err != nil {
		return nil, err
	}

	p.start()

	return p, nil
}

func SetManager(mgr Manager) func(*Peer) {
	return func(p *Peer) {
		p.mgr = mgr
	}
}

func SetLogger(log logger.Logger) func(*Peer) {
	return func(p *Peer) {
		p.log = log
	}
}

func SetNetwork(network wire.BitcoinNet) func(*Peer) {
	return func(p *Peer) {
		p.network = network
	}
}

func SetVersion(version uint32) func(*Peer) {
	return func(p *Peer) {
		p.version = version
	}
}

func SetNonce(nonce uint64) func(*Peer) {
	return func(p *Peer) {
		p.nonce = nonce
	}
}

func SetAddress(addr *net.TCPAddr) func(*Peer) {
	return func(p *Peer) {
		p.addr = addr
	}
}

func SetConnection(conn *net.TCPConn) func(*Peer) {
	return func(p *Peer) {
		p.conn = conn
	}
}

func (p *Peer) String() string {
	return p.addr.String()
}

func (p *Peer) Addr() *net.TCPAddr {
	return p.addr
}

func (p *Peer) Stop() {
	p.shutdown()
	p.wg.Wait()
}

func (p *Peer) connect() {
	// if we can't establish the connection, abort
	connGen, err := net.DialTimeout("tcp", p.addr.String(), timeoutDial)
	if err != nil {
		p.shutdown()
		return
	}

	// this should always work
	conn, ok := connGen.(*net.TCPConn)
	if !ok {
		p.shutdown()
		return
	}

	p.conn = conn

	// this should also always work...
	err = p.parse()
	if err != nil {
		p.shutdown()
		return
	}

	p.start()
}

func (p *Peer) parse() error {
	addr, ok := p.conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Could not parse remote address from connection")
	}

	you, err := wire.NewNetAddress(addr, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	local, ok := p.conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("Could not parse local address from connection")
	}

	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	p.addr = addr
	p.you = you
	p.me = me

	return nil
}

func (p *Peer) start() {
	p.wg.Add(3)
	go p.goSend()
	go p.goReceive()
	go p.goMessages()
}

func (p *Peer) shutdown() {
	// if we already called shutdown, we don't need to do it twice
	if atomic.SwapUint32(&p.done, 1) == 1 {
		return
	}

	// if we have a connection established, close it
	if p.conn != nil {
		p.conn.Close()
	}

	// signal all handlers to stop
	close(p.sigSend)
	close(p.sigRecv)
	close(p.sigMsgs)

	p.mgr.Stopped(p)
}

// sendMessage will attempt to write a message on our connection. It will set the write
// deadline in order to respect the timeout defined in our configuration. It will return
// the error if we didn't succeed.
func (p *Peer) sendMessage(msg wire.Message) error {
	p.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(p.conn, msg, p.version, p.network)

	return err
}

// recvMessage will attempt to read a message from our connection. It will set the read
// deadline in order to respect the timeout defined in our configuration. It will return
/// the read message as well as the error.
func (p *Peer) recvMessage() (wire.Message, error) {
	p.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	msg, _, err := wire.ReadMessage(p.conn, p.version, p.network)

	return msg, err
}

// handleSend is the handler responsible for sending messages in the queue. It will
// send messages from the queue and push ping messages if the connection is idling.
func (p *Peer) goSend() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	// initialize the idle timer to see when we didn't send for a while
	idleTimer := time.NewTimer(timeoutPing)

	for atomic.LoadUint32(&p.done) == 0 {
		select {
		// signal for shutdown, so break outer loop
		case _, ok := <-p.sigSend:
			if !ok {
				break
			}

		// we didnt's send a message in a long time, so send a ping
		case <-idleTimer.C:
			p.sendQ <- p.createPingMsg()

		// try to send the next message in the queue
		// on timeouts, we skip to next one, on other errors, we stop the p
		case msg := <-p.sendQ:
			err := p.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil {
				p.shutdown()
			}

			// we successfully sent a message, so reset the idle timer
			idleTimer.Reset(timeoutPing)
		}
	}
}

// handleReceive is the handler responsible for receiving messages and pushing them onto
// the reception queue. It does not do any processing so that we read all messages as quickly
// as possible.
func (p *Peer) goReceive() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	// initialize the timer to see when we didn't receive in a long time
	idleTimer := time.NewTimer(timeoutIdle)

	for atomic.LoadUint32(&p.done) == 0 {
		select {
		// the p has shutdown so break outer loop
		case _, ok := <-p.sigRecv:
			if !ok {
				break
			}

		// we didn't receive a message for too long, so time this p out and dump
		case <-idleTimer.C:
			p.shutdown()

		// each iteration without other action, we try receiving a message for a while
		// if we time out, we try again, on error we quit
		default:
			msg, err := p.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil {
				p.shutdown()
			}

			// we successfully received a message, so reset the idle timer and push it
			// onte the reception queue for further processing
			idleTimer.Reset(timeoutIdle)
			p.recvQ <- msg
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (p *Peer) goMessages() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	for atomic.LoadUint32(&p.done) == 0 {
		select {
		// shutdown signal, break outer loop
		case _, ok := <-p.sigMsgs:
			if !ok {
				break
			}

		// we read a message from the queue, process it depending on type
		case msg := <-p.recvQ:
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

func (p *Peer) handleVersionMsg(msg *wire.MsgVersion) {
	p.log.Debug("%v: received version message", p)

	if msg.Nonce == p.nonce {
		p.log.Warning("%v: detected connection to self, disconnecting", p)
		p.shutdown()
		return
	}

	/*if p.shaked {
		p.log.Notice("%v: received version after handshake", p)
		return
	}*/

	if msg.ProtocolVersion < int32(wire.MultipleAddressVersion) {
		p.log.Notice("%v: detected outdated protocol version", p)
	}

	p.version = util.MinUint32(p.version, uint32(msg.ProtocolVersion))
	//p.shaked = true

	/*if p.incoming {
		p.pushVersion()
	}

	p.pushVerAck()*/
}

func (p *Peer) createVersionMsg() *wire.MsgVersion {
	msg := wire.NewMsgVersion(p.me, p.you, p.nonce, 0)
	msg.AddUserAgent(agentName, agentVersion)
	msg.AddrYou.Services = wire.SFNodeNetwork
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(wire.RejectVersion)

	return msg
}

func (p *Peer) createVerAckMsg() *wire.MsgVerAck {
	msg := wire.NewMsgVerAck()

	return msg
}

func (p *Peer) createGetAddrMsg() *wire.MsgGetAddr {
	msg := wire.NewMsgGetAddr()

	return msg
}

func (p *Peer) createPingMsg() *wire.MsgPing {
	msg := wire.NewMsgPing(p.nonce)

	return msg
}

func (p *Peer) createPongMsg(nonce uint64) *wire.MsgPong {
	msg := wire.NewMsgPong(nonce)

	return msg
}
