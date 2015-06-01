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
	stateIdle = iota
	stateConnected
	stateRunning
	stateBusy
	stateShutdown
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

// Peer represents a single peer that we communicate with on the network. It
// groups together all necessary parameters, as well as queues and communication
// functions.
type Peer struct {
	wg         *sync.WaitGroup
	sigSend    chan struct{}
	sigRecv    chan struct{}
	sigProcess chan struct{}
	sendQ      chan wire.Message
	recvQ      chan wire.Message

	log  adaptor.Log
	mgr  adaptor.Manager
	recs []adaptor.Filter
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

// New creates a new Peer withthe given options. If required options are missing
// it returns an error as second return value.
func New(options ...func(*Peer)) (*Peer, error) {
	p := &Peer{
		wg:         &sync.WaitGroup{},
		sigSend:    make(chan struct{}),
		sigRecv:    make(chan struct{}),
		sigProcess: make(chan struct{}),
		sendQ:      make(chan wire.Message, 1),
		recvQ:      make(chan wire.Message, 1),

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

// SetLogger injects the logger to be used for logging.
func SetLog(log adaptor.Log) func(*Peer) {
	return func(p *Peer) {
		p.log = log
	}
}

// SetManager injects the manager to notify about relevant changes to status.
func SetManager(mgr adaptor.Manager) func(*Peer) {
	return func(p *Peer) {
		p.mgr = mgr
	}
}

// SetRecorder injects the recorder to be used to log events on this connection.
func SetRecorders(recs []adaptor.Filter) func(*Peer) {
	return func(p *Peer) {
		p.recs = recs
	}
}

// SetRepository injects the repository to notify about newly discovered peers.
func SetRepository(repo adaptor.Repository) func(*Peer) {
	return func(p *Peer) {
		p.repo = repo
	}
}

// SetNetwork sets the type of the network we communicate on. It can be the
// Bitcoin main network or one of the test networks.
func SetNetwork(network wire.BitcoinNet) func(*Peer) {
	return func(p *Peer) {
		p.network = network
	}
}

// SetVersion sets the maximum supported Bitcoin protocol version that we will
// use to communicate with this peer.
func SetVersion(version uint32) func(*Peer) {
	return func(p *Peer) {
		p.version = version
	}
}

// SetNonce sets the nonce that we use to detect connections to self.
func SetNonce(nonce uint64) func(*Peer) {
	return func(p *Peer) {
		p.nonce = nonce
	}
}

// SetAddress sets the address that we will try to connect to if no connection
// has been established yet.
func SetAddress(addr *net.TCPAddr) func(*Peer) {
	return func(p *Peer) {
		p.addr = addr
	}
}

// SetConnection sets an established TCP connection that this peer will use
// for his handshake.
func SetConnection(conn *net.TCPConn) func(*Peer) {
	return func(p *Peer) {
		p.conn = conn
	}
}

// String returns the address of this peer as string value.
func (p *Peer) String() string {
	return p.addr.String()
}

// Addr returns the TCP address of this peer.
func (p *Peer) Addr() *net.TCPAddr {
	return p.addr
}

// Connect will try to start a connection attempt in a non-blocking manner.
func (p *Peer) Connect() {
	go p.connect()
}

// Start will try to start the peer sub-routines in a non-blocking manner.
func (p *Peer) Start() {
	go p.start()
}

// Stop will try to initialize peer shutdown in a non-blocking manner.
func (p *Peer) Stop() {
	go p.shutdown()
}

// Greet will queue a greeting message to this peer, used to conform to the
// protocol.
func (p *Peer) Greet() {
	go p.pushVersion()
}

// Poll will queue a polling message to this peer, used to discover more peers.
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

	connGen, err := net.DialTimeout("tcp", p.addr.String(), timeoutDial)
	if err != nil {
		p.log.Debug("[PEER] %v connection failed (%v)", p, err)
		p.shutdown()
		return
	}

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

	close(p.sigRecv)
	close(p.sigProcess)
	close(p.sigSend)

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

func (p *Peer) sendMessage(msg wire.Message) error {
	p.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	version := atomic.LoadUint32(&p.version)
	err := wire.WriteMessage(p.conn, msg, version, p.network)

	return err
}

func (p *Peer) recvMessage() (wire.Message, error) {
	p.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	version := atomic.LoadUint32(&p.version)
	msg, _, err := wire.ReadMessage(p.conn, version, p.network)

	return msg, err
}

func (p *Peer) goSend() {
	defer p.wg.Done()

	p.log.Debug("[PEER] %v send routine started", p)

	idleTimer := time.NewTimer(timeoutPing)

SendLoop:
	for {
		select {
		case _, ok := <-p.sigSend:
			if !ok {
				break SendLoop
			}

		case <-idleTimer.C:
			p.pushPing()

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

func (p *Peer) goReceive() {
	defer p.wg.Done()

	p.log.Debug("[PEER] %v receive routine started", p)

	idleTimer := time.NewTimer(timeoutIdle)

ReceiveLoop:
	for {
		select {
		case _, ok := <-p.sigRecv:
			if !ok {
				break ReceiveLoop
			}

		case <-idleTimer.C:
			break ReceiveLoop

		default:
			msg, err := p.recvMessage()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			if _, ok := err.(*wire.MessageError); ok {
				p.log.Notice("[PEER] %v: received ignored (%v)", p, err)
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
		case _, ok := <-p.sigProcess:
			if !ok {
				break ProcessLoop
			}

		case msg := <-p.recvQ:
			p.processMessage(msg)
		}
	}

	timer := time.NewTimer(timeoutDrain)

DrainRecvLoop:
	for {
		select {
		case <-timer.C:
			break DrainRecvLoop

		case <-p.recvQ:
			p.log.Debug("[PEER] %v drained recv message", p)
		}
	}
}

func (p *Peer) processMessage(msg wire.Message) {
	ra, ok1 := p.conn.RemoteAddr().(*net.TCPAddr)
	la, ok2 := p.conn.LocalAddr().(*net.TCPAddr)
	if ok1 && ok2 {
		for _, rec := range p.recs {
			rec.Message(msg, ra, la)
		}
	}

	if atomic.LoadUint32(&p.rcvd) == 0 {
		_, ok := msg.(*wire.MsgVersion)
		if !ok {
			p.log.Warning("%v: out of order non-version message", p)
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

		version := atomic.LoadUint32(&p.version)
		version = util.MinUint32(version, uint32(m.ProtocolVersion))
		atomic.StoreUint32(&p.version, version)

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
