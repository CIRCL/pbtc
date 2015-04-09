package all

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

// Manager is the module responsible for managing the connections to peers and
// keep them in line with application level state and requirements. It accepts
// inbound connections, establishes the desired number of outgoing connections
// and manages the creation and disposal of peers. It will use a provided
// repository to get addresses to connect to and notifies it about changes
// relevant to address selection.
type Manager struct {
	logger      *LogHelper
	repo        *Repository
	peerIndex   map[string]*peer
	listenIndex map[string]*net.TCPListener
	sigPeer     chan struct{}
	sigConn     chan struct{}
	peerNew     chan *peer
	peerDone    chan *peer
	connTicker  *time.Ticker
	wg          *sync.WaitGroup
	state       uint32
	network     wire.BitcoinNet
	version     uint32
	nonce       uint64
}

// NewManager returns a new manager with all necessary variables initialized.
func NewManager() *Manager {
	mgr := &Manager{
		logger:     GetLogHelper("[MGR]"),
		peerIndex:  make(map[string]*peer),
		sigPeer:    make(chan struct{}, 1),
		sigConn:    make(chan struct{}, 1),
		peerNew:    make(chan *peer, bufferManagerNew),
		peerDone:   make(chan *peer, bufferManagerDone),
		connTicker: time.NewTicker(time.Second / maxConnsPerSec),
		wg:         &sync.WaitGroup{},
		state:      stateIdle,
	}

	return mgr
}

// Start starts the manager, with run-time options passed in as parameters. This allows
// us to stop and restart the manager with a different protocol version, network or even
// repository of nodes.
func (mgr *Manager) Start(repo *Repository, network wire.BitcoinNet, version uint32) {
	// we can only start the manager if it is in idle state and ready to be started
	if !atomic.CompareAndSwapUint32(&mgr.state, stateIdle, stateBusy) {
		mgr.logger.Logln(LogWarning, "Cannot start from non-idle state")
		return
	}

	// set the parameters for the nodes and connections we will create
	mgr.logger.Logln(LogTrace, "Assigning configuration parameters")
	mgr.repo = repo
	mgr.network = network
	mgr.version = version

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	mgr.wg.Add(3)
	go mgr.handleListeners()
	go mgr.handleConnections()
	go mgr.handlePeers()

	// at this point, start-up is complete and we can set the new state
	atomic.StoreUint32(&mgr.state, stateRunning)
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *Manager) Stop() {
	// we can only stop the manager if we are currently in running state
	if !atomic.CompareAndSwapUint32(&mgr.state, stateRunning, stateBusy) {
		return
	}

	// first we will stop every peer - this is a blocking operation
	for _, peer := range mgr.peerIndex {
		peer.Stop()
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

	// we then wait for all handlers to finish cleanly
	mgr.wg.Wait()

	// at this point, all handlers have stopped and we are back in idle state
	atomic.StoreUint32(&mgr.state, stateIdle)
}

// handleListeners tries to start a listener on every local IP to accept
// connections. It should be called as a go routine.
func (mgr *Manager) handleListeners() {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()

	// get all IPs on local interfaces and iterate through them
	ips := FindLocalIPs()
	for _, ip := range ips {
		// if we can't convert into a TCP address, skip
		addr, err := net.ResolveTCPAddr("tcp", ip.String())
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
		go mgr.processListener(listener)
	}
}

// handleConnections attempts to establish new connections at the configured
// rate as long as we are not at the maximum number of connections.
func (mgr *Manager) handleConnections() {
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
			if len(mgr.peerIndex) <= maxPeerCount {
				mgr.addPeer()
			}
		}
	}
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (mgr *Manager) handlePeers() {
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
		case peer := <-mgr.peerNew:
			mgr.processNewPeer(peer)

		// whenever there is an expired peer to be removed, process it
		case peer := <-mgr.peerDone:
			mgr.processDonePeer(peer)
		}
	}
}

// addPeer will try to connect to a new peer and start it on success.
func (mgr *Manager) addPeer() {
	tries := 0
	var addr *net.TCPAddr
	for {
		// if we tried too many times, give up for this time
		tries++
		if tries > maxAddrAttempts {
			return
		}

		// try to get the best address from the repository
		addr, err := mgr.repo.Get()
		if err != nil {
			return
		}

		// check if the address in still unused
		_, ok := mgr.peerIndex[addr.String()]
		if !ok {
			continue
		}

		// at this point we have a good address and can break from the loop
		break
	}

	// we initialize a new peer which will callback through a channel on success
	err := newOutgoingPeer(mgr, addr, mgr.network, mgr.version, mgr.nonce)
	if err != nil {
		return
	}
}

// processNewPeer is what we do with new initialized peers that are added to
// the manager. The peers should be in a connected state so we can start them
// and add them to our index.
func (mgr *Manager) processNewPeer(peer *peer) {
	peer.Start()

	mgr.peerIndex[peer.String()] = peer
}

// processDonePeer is what we do to expired peers. They failed in some way and
// already initialized shutdown on their own, so we just need to remove them
// from our index.
func (mgr *Manager) processDonePeer(peer *peer) {
	_, ok := mgr.peerIndex[peer.String()]
	if !ok {
		return
	}

	delete(mgr.peerIndex, peer.String())
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (mgr *Manager) processListener(listener *net.TCPListener) {
	// let the waitgroup know when we are done
	defer mgr.wg.Done()

	for {
		// try accepting a new connection
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		// create a new incoming peer for the given connection
		// if the connection is valid, the peer will notify the manager on its own
		err = newIncomingPeer(mgr, conn, mgr.network, mgr.version, mgr.nonce)
		if err != nil {
			continue
		}
	}
}
