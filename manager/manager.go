package manager

import (
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
	"github.com/CIRCL/pbtc/peer"
	"github.com/CIRCL/pbtc/util"
)

// Manager is the module responsible for peer management. It will initialize
// new incoming & outgoing peers and take care of state transitions. As the
// main control instance, it defines most of the behaviour of our peer.
type Manager struct {
	log     adaptor.Log
	peerLog adaptor.Log
	repo    adaptor.Repository
	recs    []adaptor.Processor

	wg *sync.WaitGroup

	peerSig       chan struct{}
	addrSig       chan struct{}
	addrQ         chan *net.TCPAddr
	connQ         chan *net.TCPConn
	peerConnected chan adaptor.Peer
	peerReady     chan adaptor.Peer
	peerStopped   chan adaptor.Peer

	peerIndex   *parmap.ParMap
	invIndex    *parmap.ParMap
	listenIndex map[string]*net.TCPListener

	addrTicker *time.Ticker
	infoTicker *time.Ticker

	network     wire.BitcoinNet
	version     uint32
	nonce       uint64
	connRate    time.Duration
	infoRate    time.Duration
	peerLimit   int
	defaultPort int

	done   uint32
	server bool
}

// New returns a new manager initialized with the given options.
func New(options ...func(mgr *Manager)) (*Manager, error) {
	mgr := &Manager{
		wg: &sync.WaitGroup{},

		peerSig:       make(chan struct{}),
		addrSig:       make(chan struct{}),
		addrQ:         make(chan *net.TCPAddr, 1),
		connQ:         make(chan *net.TCPConn, 1),
		peerConnected: make(chan adaptor.Peer, 1),
		peerReady:     make(chan adaptor.Peer, 1),
		peerStopped:   make(chan adaptor.Peer, 1),

		peerIndex:   parmap.New(),
		invIndex:    parmap.New(),
		listenIndex: make(map[string]*net.TCPListener),

		network:     wire.TestNet3,
		version:     wire.RejectVersion,
		defaultPort: 18333,
		infoRate:    time.Second * 5,
		connRate:    time.Second / 10,
		peerLimit:   100,
	}

	mgr.nonce, _ = wire.RandomUint64()

	for _, option := range options {
		option(mgr)
	}

	mgr.addrTicker = time.NewTicker(mgr.connRate)
	mgr.infoTicker = time.NewTicker(mgr.infoRate)

	switch mgr.network {
	case wire.TestNet3:
		mgr.defaultPort = 18333

	case wire.MainNet:
		mgr.defaultPort = 8333
	}

	if mgr.server {
		mgr.createListeners()
	}

	mgr.wg.Add(2)
	go mgr.goPeers()
	go mgr.goAddresses()

	return mgr, nil
}

// SetLog has to be passed as a parameter on manager creation. It injects
// the log to be used for logging.
func SetLog(log adaptor.Log) func(*Manager) {
	return func(mgr *Manager) {
		mgr.log = log
	}
}

// SetPeerLog has to be passed as a parameter on manager creation. It injects
// the log to be used by created peers for logging.
func SetPeerLog(log adaptor.Log) func(*Manager) {
	return func(mgr *Manager) {
		mgr.peerLog = log
	}
}

// SetRepository has to be passed as a parameter on manager creation. It injects
// the repository to be used for node management.
func SetRepository(repo adaptor.Repository) func(*Manager) {
	return func(mgr *Manager) {
		mgr.repo = repo
	}
}

// SetNetwork has to be passed as a parameter on manager creation. It sets the
// Bitcoin network to be used (main, test, regression, ...).
func SetNetwork(network wire.BitcoinNet) func(*Manager) {
	return func(mgr *Manager) {
		mgr.network = network
	}
}

// SetVersion has to be passed as a parameter on manager creation. It sets the
// maximum protocol version to be used for peer communication.
func SetVersion(version uint32) func(*Manager) {
	return func(mgr *Manager) {
		mgr.version = version
	}
}

// SetConnectionRate has to be passed as a parameter on manager creation. It
// sets the maximum number of attempted TCP connections per second.
func SetConnectionRate(connRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.connRate = connRate
	}
}

// SetInformationRate has to be passed as a parameter on manager creation. It
// defines the rate at which manager information will be output to the log.
func SetInformationRate(infoRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.infoRate = infoRate
	}
}

// SetPeerLimit has to be passed as a parameter on manager creation. It sets
// the maximum number of concurrent TCP connections, thus limiting the total
// number of connecting and connected peers.
func SetPeerLimit(peerLimit int) func(*Manager) {
	return func(mgr *Manager) {
		mgr.peerLimit = peerLimit
	}
}

// EnableServer has to be passed as a parameter on manager creation. It enables
// listening on all connected TCP IP interfaces for incoming peers.
func EnableServer() func(*Manager) {
	return func(mgr *Manager) {
		mgr.server = true
	}
}

// AddFilter has to be passed as a parameter on manager creation. It adds a
// filter for incoming messages which receives all messages for filtering and
// further forwarding.
func SetProcessors(processors ...adaptor.Processor) func(*Manager) {
	return func(mgr *Manager) {
		mgr.recs = processors
	}
}

// Close will clean-up before shutdown.
func (mgr *Manager) Close() {
	if atomic.SwapUint32(&mgr.done, 1) == 1 {
		return
	}

	close(mgr.addrSig)

	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	close(mgr.peerSig)

	for s := range mgr.peerIndex.Iter() {
		p := s.(adaptor.Peer)
		p.Stop()
	}

	mgr.wg.Wait()
}

// Connected signals to the manager that we have successfully established a
// TCP connection to a peer.
func (mgr *Manager) Connected(p adaptor.Peer) {
	mgr.peerConnected <- p
}

// Ready signals to the manager that we have successfully completed the Bitcoin
// protocol handshake with a peer.
func (mgr *Manager) Ready(p adaptor.Peer) {
	mgr.peerReady <- p
}

// Stopped signals to the manager that the connection to this peer has been
// shut down.
func (mgr *Manager) Stopped(p adaptor.Peer) {
	mgr.peerStopped <- p
}

// Knows asks the manager if it knows about a certain item on the Bitcoin
// network already. It is used to cut down on the number of redundant requests
// and logging.
func (mgr *Manager) Knows(hash wire.ShaHash) bool {
	return mgr.invIndex.Has(hash)
}

// Mark lets the manager known that a certain item has been seen and does not
// need to be requested or logged again.
func (mgr *Manager) Mark(hash wire.ShaHash) {
	mgr.invIndex.Insert(hash)
}

// creates a listener for each local IP on each local interface
func (mgr *Manager) createListeners() {
	ips, err := util.FindLocalIPs()
	if err != nil {
		return
	}

	for _, ip := range ips {
		addr := &net.TCPAddr{IP: ip, Port: mgr.defaultPort}

		_, ok := mgr.listenIndex[addr.String()]
		if ok {
			continue
		}

		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			continue
		}

		mgr.listenIndex[addr.String()] = listener
		mgr.wg.Add(1)
		go mgr.goConnections(listener)
	}
}

// to be called from a go routine
// will request and receive addresses for our connection attempts
func (mgr *Manager) goAddresses() {
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Address routine started")

AddressLoop:
	for {
		select {
		case _, ok := <-mgr.addrSig:
			if !ok {
				break AddressLoop
			}

		// at the pace of addr ticker, we request addresses to connect to
		// as long as we have not reached the peer limit
		case <-mgr.addrTicker.C:
			if mgr.peerIndex.Count() >= mgr.peerLimit {
				continue
			}

			mgr.repo.Retrieve(mgr.addrQ)
		}
	}

	mgr.log.Info("[MGR] Address routine stopped")
}

// to be called from a go routine
// will try to accept incoming peers on the given listener
func (mgr *Manager) goConnections(listener *net.TCPListener) {
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Connection routine started (%v)", listener.Addr())

	for {
		conn, err := listener.AcceptTCP()
		// unfortunately, listener does not follow the convention of returning
		// an io.EOF on closed connection, so we need to find out like this
		if err != nil &&
			strings.Contains(err.Error(), "use of closed network connection") {
			break
		}
		if err != nil {
			mgr.log.Warning("[MGR] %v: could not accept connection (%v)",
				listener.Addr(), err)
			break
		}

		// we are only interested in TCP connections (should never fail)
		addr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			conn.Close()
			break
		}

		// only accept connections to port 8333 for now (for easy counting)
		if addr.Port != 8333 {
			conn.Close()
			break
		}

		// we submit the connetion for peer creation
		mgr.connQ <- conn
	}

	mgr.log.Info("[MGR] Connection routine stopped (%v)", listener.Addr())
}

// to be called from a go routine
// will manage all peer connection/disconnection
func (mgr *Manager) goPeers() {
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Peer routine started")

PeerLoop:
	for {
		select {
		case _, ok := <-mgr.peerSig:
			if !ok {
				break PeerLoop
			}

		// print manager information to the log
		case <-mgr.infoTicker.C:
			mgr.log.Info("[MGR] %v total peers managed", mgr.peerIndex.Count())

		// create new outgoing peers for received addresses
		case addr := <-mgr.addrQ:
			if mgr.peerIndex.HasKey(addr.String()) {
				mgr.log.Debug("[MGR] %v already created", addr)
				continue
			}

			if mgr.peerIndex.Count() >= mgr.peerLimit {
				mgr.log.Debug("[MGR] %v discarded, limit reached", addr)
				continue
			}

			p, err := peer.New(
				peer.SetLog(mgr.peerLog),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetRecorders(mgr.recs),
				peer.SetNetwork(mgr.network),
				peer.SetVersion(mgr.version),
				peer.SetNonce(mgr.nonce),
				peer.SetAddress(addr),
			)
			if err != nil {
				mgr.log.Error("[MGR] %v failed outbound (%v)", addr, err)
				continue
			}

			mgr.log.Debug("[MGR] %v created", p)
			mgr.peerIndex.Insert(p)
			mgr.repo.Attempted(p.Addr())
			p.Connect()

		// create new incoming peer for received connections
		case conn := <-mgr.connQ:
			addr := conn.RemoteAddr()
			if mgr.peerIndex.HasKey(addr.String()) {
				mgr.log.Notice("[MGR] limit reached, %v not accepted", addr)
				conn.Close()
				continue
			}

			if mgr.peerIndex.Count() >= mgr.peerLimit {
				mgr.log.Debug("[MGR] %v disconnected, limit reached", addr)
				conn.Close()
				continue
			}

			p, err := peer.New(
				peer.SetLog(mgr.peerLog),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetRecorders(mgr.recs),
				peer.SetNetwork(mgr.network),
				peer.SetVersion(mgr.version),
				peer.SetNonce(mgr.nonce),
				peer.SetConnection(conn),
			)
			if err != nil {
				mgr.log.Error("[MGR] %v failed inbound (%v)", addr, err)
				continue
			}

			mgr.log.Debug("[MGR] %v accepted", p)
			mgr.peerIndex.Insert(p)
			mgr.repo.Attempted(p.Addr())
			mgr.repo.Connected(p.Addr())
			p.Start()

		// manage peers that have successfully connected
		case p := <-mgr.peerConnected:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v connected unknown", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v connected", p)
			mgr.repo.Connected(p.Addr())
			p.Start()
			p.Greet()

		// manage peers that have completed the handshake
		case p := <-mgr.peerReady:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v already ready", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v ready", p)
			mgr.repo.Succeeded(p.Addr())
			p.Poll()

		// manage peers that have dropped the connection
		case p := <-mgr.peerStopped:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v done unknown", p)
				continue
			}

			mgr.log.Debug("[MGR] %v: done", p)
			mgr.peerIndex.Remove(p)
		}
	}

	// wait for all peers to stop and drain the channels
	for mgr.peerIndex.Count() > 0 {
		select {
		case <-mgr.addrQ:
			break

		case conn := <-mgr.connQ:
			conn.Close()
			break

		case p := <-mgr.peerConnected:
			p.Stop()
			break

		case p := <-mgr.peerReady:
			p.Stop()
			break

		case p := <-mgr.peerStopped:
			mgr.peerIndex.Remove(p)
			break
		}
	}

	mgr.log.Info("[MGR] Peer routine stopped")
}
