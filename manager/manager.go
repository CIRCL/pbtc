package manager

import (
	"net"
	"strconv"
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

// Manager is the module responsible for managing the connections to peers and
// keep them in line with application level state and requirements. It accepts
// inbound connections, establishes the desired number of outgoing connections
// and manages the creation and disposal of peers. It will use a provided
// repository to get addresses to connect to and notifies it about changes
// relevant to address selection.
type Manager struct {
	wg          *sync.WaitGroup
	sigPeer     chan struct{}
	sigConn     chan struct{}
	peerStarted chan adaptor.Peer
	peerReady   chan adaptor.Peer
	peerStopped chan adaptor.Peer
	connTicker  *time.Ticker
	peerIndex   map[string]adaptor.Peer
	listenIndex map[string]*net.TCPListener

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
		wg:          &sync.WaitGroup{},
		sigPeer:     make(chan struct{}, 1),
		sigConn:     make(chan struct{}, 1),
		peerStarted: make(chan adaptor.Peer, 1),
		peerReady:   make(chan adaptor.Peer, 1),
		peerStopped: make(chan adaptor.Peer, 1),
		connTicker:  time.NewTicker(time.Second / 4),
		peerIndex:   make(map[string]adaptor.Peer),
		listenIndex: make(map[string]*net.TCPListener),

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

func (mgr *Manager) Started(p adaptor.Peer) {
	mgr.peerStarted <- p
}

func (mgr *Manager) Ready(p adaptor.Peer) {
	mgr.peerReady <- p
}

func (mgr *Manager) Stopped(p adaptor.Peer) {
	mgr.peerStopped <- p
}

// Start starts the manager, with run-time options passed in as parameters. This allows
// us to stop and restart the manager with a different protocol version, network or even
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
	for _, peer := range mgr.peerIndex {
		peer.Cleanup()
	}

	// here, we close the channel to signal the connection handler to stop
	close(mgr.sigConn)

	// the listener handler already quits after launching all listeners
	// we thus only need to close all listeners and wait for their routines to stop
	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	// finally, we signal the peer listener to stop processing as well
	close(mgr.sigPeer)
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
		addr, err := net.ResolveTCPAddr("tcp", ip.String()+":"+strconv.Itoa(18333))
		if err != nil {
			continue
		}

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
			if len(mgr.peerIndex) < 128 {
				mgr.addPeer()
			}
		}
	}
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (mgr *Manager) goPeers() {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()

PeerLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mgr.sigPeer:
			if !ok {
				break PeerLoop
			}

		// whenever there is a new peer to be added, process it
		case p := <-mgr.peerStarted:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				p.Cleanup()
				continue
			}

			mgr.repo.Attempted(p.Addr())

		case p := <-mgr.peerReady:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				p.Cleanup()
				continue
			}

			mgr.repo.Succeeded(p.Addr())

		// whenever there is an expired peer to be removed, process it
		case p := <-mgr.peerStopped:
			_, ok := mgr.peerIndex[p.String()]
			if !ok {
				p.Cleanup()
				continue
			}

			delete(mgr.peerIndex, p.String())
		}
	}
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (mgr *Manager) handleListener(listener *net.TCPListener) {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()

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
			break
		}

		// create a new incoming peer for the given connection
		// if the connection is valid, the peer will notify the manager on its own
		p, err := peer.New(
			peer.SetManager(mgr),
			peer.SetRecorder(mgr.rec),
			peer.SetNetwork(mgr.network),
			peer.SetVersion(mgr.version),
			peer.SetNonce(mgr.nonce),
			peer.SetConnection(conn),
		)
		if err != nil {
			continue
		}

		_ = p
	}
}

// addPeer will try to connect to a new peer and start it on success.
func (mgr *Manager) addPeer() {

	tries := 0
	for {
		// if we tried too many times, give up for this time
		tries++
		if tries > 128 {
			return
		}

		// try to get the best address from the repository
		addr := mgr.repo.Retrieve()

		// check if the address in still unused
		_, ok := mgr.peerIndex[addr.String()]
		if ok {
			continue
		}

		// we initialize a new peer which will callback through a channel on success
		p, err := peer.New(
			peer.SetManager(mgr),
			peer.SetRecorder(mgr.rec),
			peer.SetNetwork(mgr.network),
			peer.SetVersion(mgr.version),
			peer.SetNonce(mgr.nonce),
			peer.SetAddress(addr),
		)
		if err != nil {
			return
		}

		_ = p

		mgr.repo.Attempted(addr)
		break
	}
}
