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
	sent uint32
	rcvd uint32
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

func (p *Peer) Cleanup() {
	p.shutdown()
	p.wg.Wait()
}

func (p *Peer) Poll() {
	p.pushGetAddr()
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
	p.wg.Add(2)
	go p.goSend()
	go p.goReceive()
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
			p.pushPing()

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
			p.processMessage(msg)
		}
	}
}

// handleMessages is the handler to process messages from our reception queue.
func (p *Peer) processMessage(msg wire.Message) {

	p.mgr.Message(msg)

	if atomic.LoadUint32(&p.rcvd) == 0 {
		_, ok := msg.(*wire.MsgVersion)
		if !ok {
			p.log.Warning("%v: out of order non-version message")
			p.shutdown()
			return
		}
	}

	// we read a message from the queue, process it depending on type
	switch m := msg.(type) {
	case *wire.MsgVersion:
		if m.Nonce == p.nonce {
			p.log.Warning("%v: detected connection to self")
			p.shutdown()
			return
		}

		if uint32(m.ProtocolVersion) < wire.MultipleAddressVersion {
			p.log.Warning("%v: connected to obsolete peer")
			p.shutdown()
			return
		}

		if atomic.SwapUint32(&p.rcvd, 1) == 1 {
			p.log.Warning("%v: out of order version message")
			p.shutdown()
			return
		}

		p.version = util.MinUint32(p.version, uint32(m.ProtocolVersion))

		if atomic.SwapUint32(&p.sent, 1) == 0 {
			p.pushVersion()
		}

		p.pushVerAck()
		p.pushAddr()

	case *wire.MsgVerAck:

	case *wire.MsgPing:
		p.pushPong(m.Nonce)

	case *wire.MsgPong:

	case *wire.MsgGetAddr:
		p.pushAddr()

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
	p.sendQ <- wire.NewMsgAddr()
}
