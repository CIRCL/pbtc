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
	"github.com/CIRCL/pbtc/convertor"
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

	log     adaptor.Log
	mgr     adaptor.Manager
	recs    []adaptor.Processor
	repo    adaptor.Repository
	tracker adaptor.Tracker

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

// New creates a new Peer with the given options. Communication on state is done
// directly with the injected modules and not always through the manager.
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

	// we need either an address to connect to or an established connection
	if p.addr == nil && p.conn == nil {
		return nil, errors.New("Must provide address or connection")
	}

	// if we have no connection, we don't need to parse anything
	if p.conn == nil {
		return p, nil
	}

	// if we have a connection, we will try to parse the address from it
	// a peer always will have an address associated with it
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
func SetProcessors(recs []adaptor.Processor) func(*Peer) {
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

// SetTracker sets the tracker responsible for tracking inventory items
// like transactions and blocks.
func SetTracker(tracker adaptor.Tracker) func(*Peer) {
	return func(p *Peer) {
		p.tracker = tracker
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
	go p.startup()
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

// connect will try to connect to the address of the peer, if there is not
// yet a connection that has been established
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

	// we only care about TCP connections, but this should never fail
	conn, ok := connGen.(*net.TCPConn)
	if !ok {
		p.log.Warning("[PEER] %v connection type assert failed", p)
		p.shutdown()
		return
	}

	// if the peer was stoppe while trying to connect, we can discard everything
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

func (p *Peer) startup() {
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

// try to parse the connection parameters and address from the connection
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

// sendMessage is used internally to send a message, blocking for timeout
func (p *Peer) sendMessage(msg wire.Message) error {
	p.conn.SetWriteDeadline(time.Now().Add(timeoutSend))
	version := atomic.LoadUint32(&p.version)
	err := wire.WriteMessage(p.conn, msg, version, p.network)

	return err
}

// recvMessage is used internally to receive a message; it blocks for timeout
func (p *Peer) recvMessage() (wire.Message, error) {
	p.conn.SetReadDeadline(time.Now().Add(timeoutRecv))
	version := atomic.LoadUint32(&p.version)
	msg, _, err := wire.ReadMessage(p.conn, version, p.network)

	return msg, err
}

// goSend takes care of reading the send queue and putting the messages on the
// wire
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

		// send ping if nothing was sent for a while
		case <-idleTimer.C:
			p.pushPing()

		// if we have a message in the queue, send it
		case msg := <-p.sendQ:
			err := p.sendMessage(msg)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				break SendLoop
			}
			if _, ok := err.(*wire.MessageError); ok {
				p.log.Notice("[PEER] %v: send ignored (%v)", p, err)
				continue
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

	// drain messages to be sent for a defined timespan
	// this makes sure we don't get stuck somewhere because a sender is
	// blocking and the peer has already quit
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

// goReceive handles incoming messages and queues them for processing
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

		// if we haven't received a message in a while, disconnect the peer
		case <-idleTimer.C:
			p.log.Notice("[PEER] %v: peer timed out")
			break ReceiveLoop

		// try to receive a message and put in on the receive queue
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

// goProcess processes the messages in the receive queue
// we had to separate it from the reception handler so that messages wouldn't
// start queuing directly on the os socket
func (p *Peer) goProcess() {
	defer p.wg.Done()
ProcessLoop:
	for {
		select {
		case _, ok := <-p.sigProcess:
			if !ok {
				break ProcessLoop
			}

		// get messages from the receive queue and process them
		case msg := <-p.recvQ:
			p.processMessage(msg)
		}
	}

	timer := time.NewTimer(timeoutDrain)

	// drain the receive queue for a set duration to make sure the receiving
	// loop doesn't block
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

// processMessage does basic processing of the message to be in conformity
// with the bitcoin protocol and then forwards it to the respective filters
func (p *Peer) processMessage(msg wire.Message) {
	ra, ok1 := p.conn.RemoteAddr().(*net.TCPAddr)
	la, ok2 := p.conn.LocalAddr().(*net.TCPAddr)
	if ok1 && ok2 {
		record := convertor.Message(msg, ra, la)
		for _, rec := range p.recs {
			rec.Process(record)
		}
	}

	// if we have not yet received a version message and we receive any other
	// message, the peer is breaking the protocol and we disconnect
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

	// version message is valid if we have not received one yet
	// we also try to avoid out-of-date protocol peers and connections to self
	case *wire.MsgVersion:
		if atomic.SwapUint32(&p.rcvd, 1) == 1 {
			p.log.Warning("%v: out of order version message", p)
			p.Stop()
			return
		}

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

		// synchronize our protocol version to lowest supported one
		version := atomic.LoadUint32(&p.version)
		version = util.MinUint32(version, uint32(m.ProtocolVersion))
		atomic.StoreUint32(&p.version, version)

		// send the verack message
		p.pushVerAck()

		// if we have not sent our version yet, do so
		// if we have, the handshake is now complete
		if atomic.SwapUint32(&p.sent, 1) != 1 {
			p.pushVersion()
		} else {
			p.mgr.Ready(p)
		}

	// verack messages only matter if we are waiting to finish handshake
	// if we have both received and sent version, it is complete
	case *wire.MsgVerAck:
		if atomic.LoadUint32(&p.sent) == 1 && atomic.LoadUint32(&p.rcvd) == 1 {
			p.mgr.Ready(p)
		}

	// only send a pong message if the protocol version expects it
	case *wire.MsgPing:
		if p.version >= wire.BIP0031Version {
			p.pushPong(m.Nonce)
		}

	case *wire.MsgPong:

	case *wire.MsgGetAddr:

	// if we get an address message, add the addresses to the repository
	case *wire.MsgAddr:
		for _, na := range m.AddrList {
			addr := util.ParseNetAddress(na)
			p.repo.Discovered(addr)
		}

	// if we get an inventory message, ask for the inventory
	case *wire.MsgInv:
		p.pushGetData(m)

	case *wire.MsgGetHeaders:

	case *wire.MsgHeaders:

	case *wire.MsgGetBlocks:

	// if we receive a block message, mark the block hash as known
	case *wire.MsgBlock:
		p.tracker.AddBlock(m.BlockSha())

	case *wire.MsgGetData:

	// if we receive a transaction message, mark the transaction hash as known
	case *wire.MsgTx:
		p.tracker.AddTx(m.TxSha())

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
		if inv.Type == 0 && p.tracker.KnowsBlock(inv.Hash) {
			continue
		}

		if inv.Type == 1 && p.tracker.KnowsTx(inv.Hash) {
			continue
		}

		msg.AddInvVect(inv)
	}

	p.sendQ <- msg
}
