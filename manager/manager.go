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

	log  adaptor.Logger
	repo adaptor.Repository
	rec  adaptor.Recorder

	network     wire.BitcoinNet
	version     uint32
	nonce       uint64
	connRate    time.Duration
	infoRate    time.Duration
	peerLimit   int
	defaultPort int

	done uint32
}

// NewManager returns a new manager with all necessary variables initialized.
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

	mgr.start()

	return mgr, nil
}

func SetLogger(log adaptor.Logger) func(*Manager) {
	return func(mgr *Manager) {
		mgr.log = log
	}
}

func SetRepository(repo adaptor.Repository) func(*Manager) {
	return func(mgr *Manager) {
		mgr.repo = repo
	}
}

func SetRecorder(rec adaptor.Recorder) func(*Manager) {
	return func(mgr *Manager) {
		mgr.rec = rec
	}
}

func SetNetwork(network wire.BitcoinNet) func(*Manager) {
	return func(mgr *Manager) {
		mgr.network = network
	}
}

func SetVersion(version uint32) func(*Manager) {
	return func(mgr *Manager) {
		mgr.version = version
	}
}

func SetConnectionRate(connRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.connRate = connRate
	}
}

func SetInformationRate(infoRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.infoRate = infoRate
	}
}

func SetPeerLimit(peerLimit int) func(*Manager) {
	return func(mgr *Manager) {
		mgr.peerLimit = peerLimit
	}
}

func (mgr *Manager) Stop() {
	mgr.shutdown()
	mgr.wg.Wait()

	mgr.log.Info("[MGR] Shutdown complete")
}

func (mgr *Manager) Connected(p adaptor.Peer) {
	mgr.peerConnected <- p
}

func (mgr *Manager) Ready(p adaptor.Peer) {
	mgr.peerReady <- p
}

func (mgr *Manager) Stopped(p adaptor.Peer) {
	mgr.peerStopped <- p
}

func (mgr *Manager) Knows(hash wire.ShaHash) bool {
	return mgr.invIndex.Has(hash)
}

func (mgr *Manager) Mark(hash wire.ShaHash) {
	mgr.invIndex.Insert(hash)
}

// Start starts the manager, with run-time options passed in as parameters.
// us to stop and restart the manager with a different protocol version,
// repository of nodes.
func (mgr *Manager) start() {
	// listen on local IPs for incoming peers
	mgr.createListeners()

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	mgr.wg.Add(2)
	go mgr.goPeers()
	go mgr.goAddresses()

	mgr.log.Info("[MGR] Initialization complete")
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *Manager) shutdown() {
	// we can only stop the manager if we are currently in running state
	if atomic.SwapUint32(&mgr.done, 1) == 1 {
		return
	}

	close(mgr.addrSig)

	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	for s := range mgr.peerIndex.Iter() {
		p := s.(adaptor.Peer)
		p.Stop()
	}

	for mgr.peerIndex.Count() > 0 {
		time.Sleep(100 * time.Millisecond)
	}

	close(mgr.peerSig)

	mgr.wg.Wait()
}

// createListeners tries to start a listener on every local IP to accept
// connections. It should be called as a go routine.
func (mgr *Manager) createListeners() {
	// get all IPs on local interfaces and iterate through them
	ips, err := util.FindLocalIPs()
	if err != nil {
		return
	}

	for _, ip := range ips {
		// if we can't convert into a TCP address, skip
		addr := &net.TCPAddr{IP: ip, Port: mgr.defaultPort}

		// if we are already listening on this address, skip
		_, ok := mgr.listenIndex[addr.String()]
		if ok {
			continue
		}

		// if we can't create the listener, skip
		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			continue
		}

		// add the listener to our index and start an accepting handler
		// we again need to add it to the waitgroup if we want to exit cleanly
		mgr.listenIndex[addr.String()] = listener
		mgr.wg.Add(1)
		go mgr.goConnections(listener)
	}
}

// handleConnections attempts to establish new connections at the configured
// rate as long as we are not at the maximum number of connections.
func (mgr *Manager) goAddresses() {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Address routine started")

AddressLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mgr.addrSig:
			if !ok {
				break AddressLoop
			}

		// the ticker will signal each time we can attempt a new connection
		// if we don't have too many peers yet, try to create a new one
		case <-mgr.addrTicker.C:
			if mgr.peerIndex.Count() >= mgr.peerLimit {
				continue
			}

			mgr.repo.Retrieve(mgr.addrQ)
		}
	}

	mgr.log.Info("[MGR] Address routine stopped")
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (mgr *Manager) goConnections(listener *net.TCPListener) {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Connection routine started (%v)", listener.Addr())

	for {
		// try accepting a new connection
		conn, err := listener.AcceptTCP()
		// this is ugly, but the listener does not follow the convention of
		// returning an io.EOF error, but rather an unexported one
		// we need to treat it separately to keep the logs clean, as this
		// is how we do a clean and voluntary shutdown of these handlers
		if err != nil &&
			strings.Contains(err.Error(), "use of closed network connection") {
			break
		}
		if err != nil {
			mgr.log.Warning("[MGR] %v: could not accept connection (%v)",
				listener.Addr(), err)
			break
		}

		mgr.connQ <- conn
	}

	mgr.log.Info("[MGR] Connection routine stopped (%v)", listener.Addr())
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (mgr *Manager) goPeers() {
	// let the waitgroup know when we are done
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
				mgr.log.Debug("[MGR] %v discarded, limit reached")
				continue
			}

			p, err := peer.New(
				peer.SetLogger(mgr.log),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetRecorder(mgr.rec),
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
				mgr.log.Debug("[MGR] %v disconnected, limit reached")
				conn.Close()
				continue
			}

			p, err := peer.New(
				peer.SetLogger(mgr.log),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetRecorder(mgr.rec),
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

		// whenever there is an expired peer to be removed, process it
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

	mgr.log.Info("[MGR] Peer routine stopped")
}
