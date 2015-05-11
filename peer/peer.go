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
	timeoutDial  = 1 * time.Second
	timeoutSend  = 1 * time.Second
	timeoutRecv  = 1 * time.Second
	timeoutPing  = 1 * time.Minute
	timeoutIdle  = 3 * time.Minute
	timeoutDrain = 2 * time.Second
	agentName    = "Satoshi"
	agentVersion = "0.9.3"
)

type Peer struct {
	wg    *sync.WaitGroup
	sig   chan struct{}
	sendQ chan wire.Message
	recvQ chan wire.Message

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
		wg:    &sync.WaitGroup{},
		sig:   make(chan struct{}),
		sendQ: make(chan wire.Message, 1),
		recvQ: make(chan wire.Message, 1),

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
	go p.connect()
}

func (p *Peer) Start() {
	go p.start()
}

func (p *Peer) Stop() {
	go p.shutdown()
}

func (p *Peer) Greet() {
	go p.pushVersion()
}

func (p *Peer) Poll() {
	go p.pushGetAddr()
}

func (p *Peer) connect() {
	if atomic.LoadUint32(&p.done) != 0 {
		p.log.Warning("[PEER] %v can't connect when done", p)
		return
	}

	if p.conn != nil {
		p.log.Warning("[PEER] %v already connected", p)
		return
	}

	// if we can't establish the connection, abort
	connGen, err := net.DialTimeout("tcp", p.addr.String(), timeoutDial)
	if err != nil {
		p.log.Debug("[PEER] %v connection failed (%v)", p, err)
		p.shutdown()
		return
	}

	// this should always work
	conn, ok := connGen.(*net.TCPConn)
	if !ok {
		p.log.Warning("[PEER] %v connection type assert failed", p)
		p.shutdown()
		return
	}

	if atomic.LoadUint32(&p.done) == 1 {
		p.log.Warning("[PEER] %v connection late", p)
		conn.Close()
		p.shutdown()
		return
	}

	p.conn = conn

	// this should also always work...
	err = p.parse()
	if err != nil {
		p.log.Warning("[PEER] %v connection parsing failed", p)
		p.shutdown()
		return
	}

	p.log.Debug("[PEER] %v connection established", p)
	p.mgr.Connected(p)
}

func (p *Peer) start() {
	if atomic.SwapUint32(&p.started, 1) == 1 {
		return
	}

	p.wg.Add(3)
	go p.goSend()
	go p.goReceive()
	go p.goProcess()
}

func (p *Peer) shutdown() {
	if atomic.SwapUint32(&p.done, 1) == 1 {
		return
	}

	close(p.sig)

	p.wg.Wait()

	if p.conn != nil {
		p.conn.Close()
	}

	p.mgr.Stopped(p)
}

func (p *Peer) parse() error {
	if p.addr == nil {
		return errors.New("can't parse nil address")
	}

	you, err := wire.NewNetAddress(p.addr, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	local, ok := p.conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("could not parse local address from connection")
	}

	me, err := wire.NewNetAddress(local, wire.SFNodeNetwork)
	if err != nil {
		return err
	}

	p.you = you
	p.me = me

	return nil
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

	p.log.Debug("[PEER] %v send routine started", p)

	// initialize the idle timer to see when we didn't send for a while
	idleTimer := time.NewTimer(timeoutPing)

SendLoop:
	for {
		select {
		// signal for shutdown, so break outer loop
		case _, ok := <-p.sig:
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
				break SendLoop
			}
			if err != nil && strings.Contains(err.Error(),
				"use of closed network connection") {
				break SendLoop
			}
			if err != nil {
				p.log.Warning("[PEER] %v: send failed (%v)", p, err)
				break SendLoop
			}

			// we successfully sent a message, so reset the idle timer
			idleTimer.Reset(timeoutPing)
		}
	}

	p.Stop()

	idleTimer.Reset(timeoutDrain)

DrainLoop:
	for {
		select {
		case <-idleTimer.C:
			break DrainLoop

		case <-p.sendQ:
			p.log.Debug("[PEER] %v drained message", p)
		}
	}

	p.log.Debug("[PEER] %v send routine stopped", p)
}

// handleReceive is the handler responsible for receiving messages and pushing
// the reception queue. It does not do any processing so that we read all messag
// as possible.
func (p *Peer) goReceive() {
	// let the waitgroup know when we are done
	defer p.wg.Done()

	p.log.Debug("[PEER] %v receive routine started", p)

	// initialize the timer to see when we didn't receive in a long time
	idleTimer := time.NewTimer(timeoutIdle)

ReceiveLoop:
	for {
		select {
		// the p has shutdown so break outer loop
		case _, ok := <-p.sig:
			if !ok {
				break ReceiveLoop
			}

		// we didn't receive a message for too long, so time this p out and dump
		case <-idleTimer.C:
			break ReceiveLoop

		// each iteration without other action, we try receiving a message for a
		// if we time out, we try again, on error we quit
		default:
			msg, err := p.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if err != nil && strings.Contains(err.Error(),
				"use of closed network connection") {
				break ReceiveLoop
			}
			if err != nil {
				p.log.Warning("[PEER] %v: receive failed (%v)", p, err)
				break ReceiveLoop
			}

			// we successfully received a message, so reset the idle timer and
			// onte the reception queue for further processing
			idleTimer.Reset(timeoutIdle)
			p.recvQ <- msg
		}
	}

	p.Stop()

	p.log.Debug("[PEER] %v receive routine stopped", p)
}

func (p *Peer) goProcess() {
	defer p.wg.Done()
ProcessLoop:
	for {
		select {
		case _, ok := <-p.sig:
			if !ok {
				break ProcessLoop
			}

		case msg := <-p.recvQ:
			p.processMessage(msg)
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (p *Peer) processMessage(msg wire.Message) {
	ra, ok1 := p.conn.RemoteAddr().(*net.TCPAddr)
	la, ok2 := p.conn.LocalAddr().(*net.TCPAddr)
	if ok1 && ok2 {
		p.rec.Message(msg, ra, la)
	}

	if atomic.LoadUint32(&p.rcvd) == 0 {
		_, ok := msg.(*wire.MsgVersion)
		if !ok {
			p.log.Warning("%v: out of order non-version message", p.String())
			p.Stop()
			return
		}
	}

	// we read a message from the queue, process it depending on type
	switch m := msg.(type) {
	case *wire.MsgVersion:
		if m.Nonce == p.nonce {
			p.log.Warning("%v: detected connection to self", p)
			p.Stop()
			return
		}

		if uint32(m.ProtocolVersion) < wire.MultipleAddressVersion {
			p.log.Warning("%v: connected to obsolete peer", p)
			p.Stop()
			return
		}

		if atomic.SwapUint32(&p.rcvd, 1) == 1 {
			p.log.Warning("%v: out of order version message", p)
			p.Stop()
			return
		}

		p.version = util.MinUint32(p.version, uint32(m.ProtocolVersion))

		p.pushVersion()
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
		p.pushGetData(m)

	case *wire.MsgGetHeaders:

	case *wire.MsgHeaders:

	case *wire.MsgGetBlocks:

	case *wire.MsgBlock:
		p.mgr.Mark(m.BlockSha())

	case *wire.MsgGetData:

	case *wire.MsgTx:
		p.mgr.Mark(m.TxSha())

	case *wire.MsgAlert:

	default:

	}
}

func (p *Peer) pushVerAck() {
	p.sendQ <- wire.NewMsgVerAck()
}

func (p *Peer) pushVersion() {
	if atomic.SwapUint32(&p.sent, 1) == 1 {
		return
	}

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
	na, err := wire.NewNetAddress(p.conn.LocalAddr(), wire.SFNodeNetwork)
	if err != nil {
		return
	}

	msg.AddAddress(na)
	p.sendQ <- msg
}

func (p *Peer) pushGetData(m *wire.MsgInv) {
	msg := wire.NewMsgGetData()

	for _, inv := range m.InvList {
		if p.mgr.Knows(inv.Hash) {
			continue
		}

		msg.AddInvVect(inv)
	}

	p.sendQ <- msg
}
