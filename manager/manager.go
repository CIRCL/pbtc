package manager

import (
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
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

const (
	maxPeers = 100
)

// Manager is the module responsible for managing the connections to peers and
// keep them in line with application level state and requirements. It accepts
// inbound connections, establishes the desired number of outgoing connections
// and manages the creation and disposal of peers. It will use a provided
// repository to get addresses to connect to and notifies it about changes
// relevant to address selection.
type Manager struct {
	wg            *sync.WaitGroup
	sigPeer       chan struct{}
	sigConn       chan struct{}
	peerConnected chan adaptor.Peer
	peerReady     chan adaptor.Peer
	peerStopped   chan adaptor.Peer
	connTicker    *time.Ticker
	pollTicker    *time.Ticker
	infoTicker    *time.Ticker
	peerIndex     map[string]adaptor.Peer
	listenIndex   map[string]*net.TCPListener

	log  adaptor.Logger
	repo adaptor.Repository
	rec  adaptor.Recorder

	network wire.BitcoinNet
	version uint32
	nonce   uint64

	done uint32
}

// NewManager returns a new manager with all necessary variables initialized.
func New(options ...func(mgr *Manager)) (*Manager, error) {
	mgr := &Manager{
		wg:            &sync.WaitGroup{},
		sigPeer:       make(chan struct{}, 1),
		sigConn:       make(chan struct{}, 1),
		peerConnected: make(chan adaptor.Peer, 1),
		peerReady:     make(chan adaptor.Peer, 1),
		peerStopped:   make(chan adaptor.Peer, 1),
		connTicker:    time.NewTicker(time.Second / 20),
		pollTicker:    time.NewTicker(time.Second * 15),
		infoTicker:    time.NewTicker(time.Second * 5),
		peerIndex:     make(map[string]adaptor.Peer),
		listenIndex:   make(map[string]*net.TCPListener),

		network: wire.TestNet3,
		version: wire.RejectVersion,
	}

	mgr.nonce, _ = wire.RandomUint64()

	for _, option := range options {
		option(mgr)
	}

	mgr.startup()

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

func (mgr *Manager) Cleanup() {
	mgr.shutdown()
	mgr.wg.Wait()
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

// Start starts the manager, with run-time options passed in as parameters.
// us to stop and restart the manager with a different protocol version,
// repository of nodes.
func (mgr *Manager) startup() {
	// listen on local IPs for incoming peers
	mgr.createListeners()

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	mgr.wg.Add(2)
	go mgr.goConnections()
	go mgr.goPeers()
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *Manager) shutdown() {
	// we can only stop the manager if we are currently in running state
	if atomic.SwapUint32(&mgr.done, 1) == 1 {
		return
	}

	// first we will stop every peer - this is a blocking operation
	for _, p := range mgr.peerIndex {
		p.Stop()
	}

	// here, we close the channel to signal the connection handler to stop
	close(mgr.sigConn)

	// the listener handler already quits after launching all listeners
	// we thus only need to close all listeners and wait for their routines to
	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	// finally, we signal the incoming peer handler to stop processing as well
	close(mgr.sigPeer)
}

func (mgr *Manager) numTotal() int {
	return len(mgr.peerIndex)
}

func (mgr *Manager) numPending() int {
	pending := 0

	for _, peer := range mgr.peerIndex {
		if peer.Pending() {
			pending++
		}
	}

	return pending
}

func (mgr *Manager) numConnected() int {
	connected := 0

	for _, peer := range mgr.peerIndex {
		if peer.Connected() {
			connected++
		}
	}

	return connected
}

func (mgr *Manager) numReady() int {
	ready := 0

	for _, peer := range mgr.peerIndex {
		if peer.Ready() {
			ready++
		}
	}

	return ready
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
		addr := &net.TCPAddr{IP: ip, Port: 18333}

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
		go mgr.handleListener(listener)
	}
}

// handleConnections attempts to establish new connections at the configured
// rate as long as we are not at the maximum number of connections.
func (mgr *Manager) goConnections() {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Managing incoming peers")

ConnLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mgr.sigConn:
			if !ok {
				break ConnLoop
			}

		// the ticker will signal each time we can attempt a new connection
		// if we don't have too many peers yet, try to create a new one
		case <-mgr.connTicker.C:
			if mgr.numConnected() >= maxPeers {
				continue
			}

			mgr.addPeer()

		case <-mgr.pollTicker.C:
			if mgr.repo.Polling() {
				for _, peer := range mgr.peerIndex {
					peer.Poll()
				}
			}

		case <-mgr.infoTicker.C:
			mgr.log.Info("[MGR] Total: %v Pending: %v Connected: %v Ready: %v",
				mgr.numTotal(), mgr.numPending(), mgr.numConnected(),
				mgr.numReady())
		}
	}

	mgr.log.Info("[MGR] Incoming peer manager done")
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (mgr *Manager) goPeers() {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] Managing outgoing peers")

PeerLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mgr.sigPeer:
			if !ok {
				break PeerLoop
			}

		case p := <-mgr.peerConnected:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v: connected")
			p.Start()
			mgr.repo.Connected(p.Addr())
			p.Greet()

		case p := <-mgr.peerReady:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v: ready")
			mgr.repo.Succeeded(p.Addr())

		// whenever there is an expired peer to be removed, process it
		case p := <-mgr.peerStopped:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				continue
			}

			mgr.log.Debug("[MGR] %v: done")
			delete(mgr.peerIndex, p.String())
		}
	}

	mgr.log.Info("[MGR] Outgoing peer manager done")
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (mgr *Manager) handleListener(listener *net.TCPListener) {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()
	mgr.log.Info("[MGR] %v: listener running", listener.Addr())

	for {
		// try accepting a new connection
		conn, err := listener.AcceptTCP()
		// this is ugly, but the listener does not follow the convention of
		// returning an io.EOF error, but rather an unexported one
		// we need to treat it separately to keep the logs clean, as this
		// is how we do a clean and voluntary shutdown of these handlers
		if strings.Contains(err.Error(), "use of closed network connection") {
			break
		}
		if err != nil {
			mgr.log.Warning("[MGR] %v: could not accept connection (%v)",
				listener.Addr(), err)
			break
		}

		// create a new incoming peer for the given connection
		// if the connection is valid, the peer will notify the manager
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
			mgr.log.Error("[MGR] %v: could not create incoming peer (%v)",
				conn.RemoteAddr(), err)
			continue
		}

		mgr.log.Debug("[MGR] %v: incoming peer added", p)
		mgr.peerIndex[p.String()] = p
		p.Start()
		mgr.repo.Attempted(p.Addr())
	}

	mgr.log.Info("[MGR] %v: listener done", listener.Addr())
}

// addPeer will try to connect to a new peer and start it on success.
func (mgr *Manager) addPeer() {
	var addr *net.TCPAddr

	for i := 0; i < 128; i++ {
		addr = mgr.repo.Retrieve()
		_, ok := mgr.peerIndex[addr.String()]
		if !ok {
			break
		}
	}

	if addr == nil {
		return
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
		mgr.log.Error("[MGR] %v: could not create outgoing peer (%v)", addr,
			err)
		return
	}

	if atomic.LoadUint32(&mgr.done) == 1 {
		return
	}

	mgr.log.Debug("[MGR] %v: outgoing peer added", p)
	mgr.peerIndex[p.String()] = p
	p.Connect()
	mgr.repo.Attempted(p.Addr())
}
