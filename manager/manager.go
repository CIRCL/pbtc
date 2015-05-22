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

const (
	stateIdle      = iota // initial state where module is ready to start
	stateConnected        // peer state when it is connected
	stateRunning          // module state after a module was started
	stateBusy             // state used during state changes
	stateShutdown         // irreversible shutdown of module
)

// Manager is the module responsible for managing the connections to peers and
// keep them in line with application level state and requirements. It accepts
// inbound connections, establishes the desired number of outgoing connections
// and manages the creation and disposal of peers. It will use a provided
// repository to get addresses to connect to and notifies it about changes
// relevant to address selection.
type Manager struct {
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

	log     adaptor.Log
	peerLog adaptor.Log
	repo    adaptor.Repository
	recs    []adaptor.Recorder

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

// New returns a new default initialized manager with all options applied to it
// subsequently.
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
		recs:        make([]adaptor.Recorder, 0, 2),

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

	mgr.start()

	return mgr, nil
}

// SetLogger injects a logger into the manager. It is required.
func SetLog(log adaptor.Log) func(*Manager) {
	return func(mgr *Manager) {
		mgr.log = log
	}
}

func SetPeerLog(log adaptor.Log) func(*Manager) {
	return func(mgr *Manager) {
		mgr.peerLog = log
	}
}

// SetRepository injects a node repository into the manager. It is required.
func SetRepository(repo adaptor.Repository) func(*Manager) {
	return func(mgr *Manager) {
		mgr.repo = repo
	}
}

// SetRecorder injects an event recorder into the manager. It is required.
func AddRecorder(rec adaptor.Recorder) func(*Manager) {
	return func(mgr *Manager) {
		mgr.recs = append(mgr.recs, rec)
	}
}

// SetNetwork sets the network on which the manager operatios. It can be the
// Bitcoin main network or one of the test networks.
func SetNetwork(network wire.BitcoinNet) func(*Manager) {
	return func(mgr *Manager) {
		mgr.network = network
	}
}

// SetVersion sets the Bitcoin protocol version that the manager uses to
// initialize its peers.
func SetVersion(version uint32) func(*Manager) {
	return func(mgr *Manager) {
		mgr.version = version
	}
}

// SetConnectionRate sets the maximum number of TCP connections the manager will
// try to establish per second
func SetConnectionRate(connRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.connRate = connRate
	}
}

// SetInformationRate sets the interval at which the manager will log an
// information summary.
func SetInformationRate(infoRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.infoRate = infoRate
	}
}

// SetPeerLimit sets the maximum number of peers to manage, which also puts a
// limit on the maximum number of concurrent TCP connections.
func SetPeerLimit(peerLimit int) func(*Manager) {
	return func(mgr *Manager) {
		mgr.peerLimit = peerLimit
	}
}

func EnableServer() func(*Manager) {
	return func(mgr *Manager) {
		mgr.server = true
	}
}

// Stop will shut the manager down and wait for all components to exit cleanly
// before returning.
func (mgr *Manager) Stop() {
	mgr.shutdown()
	mgr.wg.Wait()

	mgr.log.Info("[MGR] Shutdown complete")
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

// Start commences the execution of the manager sub-routines in a non-blocking
// way.
func (mgr *Manager) start() {
	// listen on local IPs for incoming peers
	if mgr.server {
		mgr.createListeners()
	}

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	mgr.wg.Add(2)
	go mgr.goPeers()
	go mgr.goAddresses()

	mgr.log.Info("[MGR] Initialization complete")
}

func (mgr *Manager) shutdown() {
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

		case <-mgr.addrTicker.C:
			if mgr.peerIndex.Count() >= mgr.peerLimit {
				continue
			}

			mgr.repo.Retrieve(mgr.addrQ)
		}
	}

	mgr.log.Info("[MGR] Address routine stopped")
}

func (mgr *Manager) goConnections(listener *net.TCPListener) {
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Connection routine started (%v)", listener.Addr())

	for {
		conn, err := listener.AcceptTCP()
		if err != nil &&
			strings.Contains(err.Error(), "use of closed network connection") {
			break
		}
		if err != nil {
			mgr.log.Warning("[MGR] %v: could not accept connection (%v)",
				listener.Addr(), err)
			break
		}

		addr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			conn.Close()
			break
		}

		if addr.Port != 8333 {
			conn.Close()
			break
		}

		mgr.connQ <- conn
	}

	mgr.log.Info("[MGR] Connection routine stopped (%v)", listener.Addr())
}

func (mgr *Manager) goPeers() {
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Peer routine started")

PeerLoop:
	for {
		select {
		case <-mgr.infoTicker.C:
			mgr.log.Info("[MGR] %v total peers managed", mgr.peerIndex.Count())

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

		case p := <-mgr.peerReady:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v already ready", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v ready", p)
			mgr.repo.Succeeded(p.Addr())
			p.Poll()

		case p := <-mgr.peerStopped:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v done unknown", p)
				continue
			}

			mgr.log.Debug("[MGR] %v: done", p)
			mgr.peerIndex.Remove(p)

		case _, ok := <-mgr.peerSig:
			if !ok {
				break PeerLoop
			}
		}
	}

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
