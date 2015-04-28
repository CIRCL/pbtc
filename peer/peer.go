package peer

import (
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
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
	agentName    = "Satoshi"
	agentVersion = "0.9.3"
)

type Peer struct {
	wg      *sync.WaitGroup
	sigSend chan struct{}
	sigRecv chan struct{}
	sendQ   chan wire.Message
	recvQ   chan wire.Message

	log  adaptor.Logger
	mgr  adaptor.Manager
	rec  adaptor.Recorder
	repo adaptor.Repository

	network wire.BitcoinNet
	version uint32
	nonce   uint64
	addr    *net.TCPAddr
	conn    *net.TCPConn
	me      *wire.NetAddress
	you     *wire.NetAddress

	started uint32
	done    uint32
	sent    uint32
	rcvd    uint32
}

func New(options ...func(*Peer)) (*Peer, error) {
	p := &Peer{
		wg:      &sync.WaitGroup{},
		sigSend: make(chan struct{}, 1),
		sigRecv: make(chan struct{}, 1),
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
		return p, nil
	}

	addr, ok := p.conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("Could not parse remote address from connection")
	}

	p.addr = addr

	err := p.parse()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func SetLogger(log adaptor.Logger) func(*Peer) {
	return func(p *Peer) {
		p.log = log
	}
}

func SetManager(mgr adaptor.Manager) func(*Peer) {
	return func(p *Peer) {
		p.mgr = mgr
	}
}

func SetRecorder(rec adaptor.Recorder) func(*Peer) {
	return func(p *Peer) {
		p.rec = rec
	}
}

func SetRepository(repo adaptor.Repository) func(*Peer) {
	return func(p *Peer) {
		p.repo = repo
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

func (p *Peer) Connect() {
	if atomic.LoadUint32(&p.done) != 0 {
		return
	}

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

	p.mgr.Connected(p)
}

func (p *Peer) Start() {
	if atomic.SwapUint32(&p.started, 1) == 1 {
		return
	}

	p.wg.Add(2)
	go p.goSend()
	go p.goReceive()
}

func (p *Peer) Greet() {
	if atomic.SwapUint32(&p.sent, 1) == 1 {
		return
	}

	p.pushVersion()
}

func (p *Peer) Stop() {
	p.shutdown()
	p.wg.Wait()
}

func (p *Peer) Poll() {
	if atomic.LoadUint32(&p.done) != 1 && atomic.LoadUint32(&p.sent) == 1 {
		p.pushGetAddr()
	}
}

func (p *Peer) Pending() bool {
	if p.conn == nil {
		return true
	}

	return false
}

func (p *Peer) Connected() bool {
	if p.conn != nil && !p.Ready() {
		return true
	}

	return false
}

func (p *Peer) Ready() bool {
	if atomic.LoadUint32(&p.sent) == 1 && atomic.LoadUint32(&p.rcvd) == 1 {
		return true
	}

	return false
}

func (p *Peer) parse() error {
	if p.addr == nil {
		return errors.New("Can't parse nil address")
	}

	you, err := wire.NewNetAddress(p.addr, wire.SFNodeNetwork)
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

	p.you = you
	p.me = me

	return nil
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

	p.mgr.Stopped(p)
}

// sendMessage will attempt to write a message on our connection. It will set th
// deadline in order to respect the timeout defined in our configuration. It wil
// the error if we didn't succeed.
func (p *Peer) sendMessage(msg wire.Message) error {
	p.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	err := wire.WriteMessage(p.conn, msg, p.version, p.network)

	return err
}

// recvMessage will attempt to read a message from our connection. It will set t
// deadline in order to respect the timeout defined in our configuration. It wil
/// the read message as well as the error.
func (p *Peer) recvMessage() (wire.Message, error) {
	p.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	msg, _, err := wire.ReadMessage(p.conn, p.version, p.network)

	return msg, err
}

// handleSend is the handler responsible for sending messages in the queue. It w
// send messages from the queue and push ping messages if the connection is idli
func (p *Peer) goSend() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	// initialize the idle timer to see when we didn't send for a while
	idleTimer := time.NewTimer(timeoutPing)

SendLoop:
	for {
		select {
		// signal for shutdown, so break outer loop
		case _, ok := <-p.sigSend:
			if !ok {
				break SendLoop
			}

		// we didnt's send a message in a long time, so send a ping
		case <-idleTimer.C:
			p.pushPing()

		// try to send the next message in the queue
		// on timeouts, we skip to next one, on other errors, we stop the p
		case msg := <-p.sendQ:
			err := p.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				p.shutdown()
				continue
			}
			if err != nil && strings.Contains(err.Error(),
				"use of closed network connection") {
				p.shutdown()
				break
			}
			if err != nil {
				p.log.Error("[PEER] %v: could not send message (%v)", p, err)
				p.shutdown()
				continue
			}

			// we successfully sent a message, so reset the idle timer
			idleTimer.Reset(timeoutPing)
		}
	}
}

// handleReceive is the handler responsible for receiving messages and pushing
// the reception queue. It does not do any processing so that we read all messag
// as possible.
func (p *Peer) goReceive() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	// initialize the timer to see when we didn't receive in a long time
	idleTimer := time.NewTimer(timeoutIdle)

ReceiveLoop:
	for {
		select {
		// the p has shutdown so break outer loop
		case _, ok := <-p.sigRecv:
			if !ok {
				break ReceiveLoop
			}

		// we didn't receive a message for too long, so time this p out and dump
		case <-idleTimer.C:
			p.shutdown()

		// each iteration without other action, we try receiving a message for a
		// if we time out, we try again, on error we quit
		default:
			msg, err := p.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil && strings.Contains(err.Error(),
				"use of closed network connection") {
				p.shutdown()
				break
			}
			if err != nil {
				p.log.Error("[PEER] %v: could not receive message (%v)", p, err)
				p.shutdown()
				continue
			}

			// we successfully received a message, so reset the idle timer and
			// onte the reception queue for further processing
			idleTimer.Reset(timeoutIdle)
			p.processMessage(msg)
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (p *Peer) processMessage(msg wire.Message) {
	la, _ := p.conn.LocalAddr().(*net.TCPAddr)
	ra, _ := p.conn.RemoteAddr().(*net.TCPAddr)
	p.rec.Message(msg, la, ra)

	if atomic.LoadUint32(&p.rcvd) == 0 {
		_, ok := msg.(*wire.MsgVersion)
		if !ok {
			p.log.Warning("%v: out of order non-version message", p.String())
			p.shutdown()
			return
		}
	}

	// we read a message from the queue, process it depending on type
	switch m := msg.(type) {
	case *wire.MsgVersion:
		if m.Nonce == p.nonce {
			p.log.Warning("%v: detected connection to self", p.String())
			p.shutdown()
			return
		}

		if uint32(m.ProtocolVersion) < wire.MultipleAddressVersion {
			p.log.Warning("%v: connected to obsolete peer", p.String())
			p.shutdown()
			return
		}

		if atomic.SwapUint32(&p.rcvd, 1) == 1 {
			p.log.Warning("%v: out of order version message", p.String())
			p.shutdown()
			return
		}

		p.version = util.MinUint32(p.version, uint32(m.ProtocolVersion))

		if atomic.SwapUint32(&p.sent, 1) == 0 {
			p.pushVersion()
		}

		p.pushVerAck()

	case *wire.MsgVerAck:
		if atomic.LoadUint32(&p.sent) == 1 && atomic.LoadUint32(&p.rcvd) == 1 {
			p.mgr.Ready(p)
		}

	case *wire.MsgPing:
		p.pushPong(m.Nonce)

	case *wire.MsgPong:

	case *wire.MsgGetAddr:

	case *wire.MsgAddr:
		for _, na := range m.AddrList {
			addr := util.ParseNetAddress(na)
			p.repo.Discovered(addr)
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

	}
}

func (p *Peer) pushVerAck() {
	p.sendQ <- wire.NewMsgVerAck()
}

func (p *Peer) pushVersion() {
	msg := wire.NewMsgVersion(p.me, p.you, p.nonce, 0)
	msg.AddUserAgent(agentName, agentVersion)
	msg.AddrYou.Services = wire.SFNodeNetwork
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(wire.RejectVersion)
	p.sendQ <- msg
}

func (p *Peer) pushPing() {
	p.sendQ <- wire.NewMsgPing(p.nonce)
}

func (p *Peer) pushPong(nonce uint64) {
	p.sendQ <- wire.NewMsgPong(nonce)
}

func (p *Peer) pushGetAddr() {
	p.sendQ <- wire.NewMsgGetAddr()
}

func (p *Peer) pushAddr() {
	msg := wire.NewMsgAddr()
	na, _ := wire.NewNetAddress(p.conn.LocalAddr(), wire.SFNodeNetwork)
	msg.AddAddress(na)
	p.sendQ <- wire.NewMsgAddr()
}
