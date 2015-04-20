package manager

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/logger"
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
	peerIndex   map[string]*peer.Peer
	listenIndex map[string]*net.TCPListener
	sigPeer     chan struct{}
	sigConn     chan struct{}
	peerStarted chan *peer.Peer
	peerReady   chan *peer.Peer
	peerStopped chan *peer.Peer
	connTicker  *time.Ticker
	wg          *sync.WaitGroup

	log  logger.Logger
	repo Repository

	network wire.BitcoinNet
	version uint32
	nonce   uint64

	done uint32
}

// NewManager returns a new manager with all necessary variables initialized.
func New(options ...func(ctr *Manager)) (*Manager, error) {
	ctr := &Manager{
		peerIndex:   make(map[string]*peer.Peer),
		listenIndex: make(map[string]*net.TCPListener),
		sigPeer:     make(chan struct{}, 1),
		sigConn:     make(chan struct{}, 1),
		peerStarted: make(chan *peer.Peer, 1),
		peerReady:   make(chan *peer.Peer, 1),
		peerStopped: make(chan *peer.Peer, 1),
		connTicker:  time.NewTicker(time.Second / 4),
		wg:          &sync.WaitGroup{},
	}

	for _, option := range options {
		option(ctr)
	}

	if ctr.network == 0 {
		ctr.network = wire.TestNet3
	}

	if ctr.version == 0 {
		ctr.version = wire.RejectVersion
	}

	if ctr.nonce == 0 {
		ctr.nonce, _ = wire.RandomUint64()
	}

	ctr.startup()

	return ctr, nil
}

func SetLogger(log logger.Logger) func(*Manager) {
	return func(ctr *Manager) {
		ctr.log = log
	}
}

func SetRepository(repo Repository) func(*Manager) {
	return func(ctr *Manager) {
		ctr.repo = repo
	}
}

func (ctr *Manager) Cleanup() {
	ctr.shutdown()
	ctr.wg.Wait()
}

func (ctr *Manager) Started(p *peer.Peer) {
	ctr.peerStarted <- p
}

func (ctr *Manager) Ready(p *peer.Peer) {
	ctr.peerReady <- p
}

func (ctr *Manager) Stopped(p *peer.Peer) {
	ctr.peerStopped <- p
}

func (ctr *Manager) Message(msg wire.Message) {

}

// Start starts the manager, with run-time options passed in as parameters. This allows
// us to stop and restart the manager with a different protocol version, network or even
// repository of nodes.
func (ctr *Manager) startup() {
	// listen on local IPs for incoming peers
	ctr.createListeners()

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	ctr.wg.Add(2)
	go ctr.goConnections()
	go ctr.goPeers()
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (ctr *Manager) shutdown() {
	// we can only stop the manager if we are currently in running state
	if atomic.SwapUint32(&ctr.done, 1) == 1 {
		return
	}

	// first we will stop every peer - this is a blocking operation
	for _, peer := range ctr.peerIndex {
		peer.Stop()
	}

	// here, we close the channel to signal the connection handler to stop
	close(ctr.sigConn)

	// the listener handler already quits after launching all listeners
	// we thus only need to close all listeners and wait for their routines to stop
	for _, listener := range ctr.listenIndex {
		listener.Close()
	}

	// finally, we signal the peer listener to stop processing as well
	close(ctr.sigPeer)
}

// createListeners tries to start a listener on every local IP to accept
// connections. It should be called as a go routine.
func (ctr *Manager) createListeners() {
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
		_, ok := ctr.listenIndex[addr.String()]
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
		ctr.listenIndex[addr.String()] = listener
		ctr.wg.Add(1)
		go ctr.handleListener(listener)
	}
}

// handleConnections attempts to establish new connections at the configured
// rate as long as we are not at the maximum number of connections.
func (ctr *Manager) goConnections() {
	// let the waitgroup know when we are done
	defer ctr.wg.Done()

ConnLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-ctr.sigConn:
			if !ok {
				break ConnLoop
			}

		// the ticker will signal each time we can attempt a new connection
		// if we don't have too many peers yet, try to create a new one
		case <-ctr.connTicker.C:
			if len(ctr.peerIndex) < 128 {
				ctr.addPeer()
			}
		}
	}
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (ctr *Manager) goPeers() {
	// let the waitgroup know when we are done
	defer ctr.wg.Done()

PeerLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-ctr.sigPeer:
			if !ok {
				break PeerLoop
			}

		// whenever there is a new peer to be added, process it
		case peer := <-ctr.peerStarted:
			ctr.repo.Connected(peer.Addr())
			ctr.repo.Succeeded(peer.Addr())

			_, ok := ctr.peerIndex[peer.String()]
			if ok {
				return
			}

			if len(ctr.peerIndex) >= 128 {
				return
			}

			ctr.peerIndex[peer.String()] = peer

		case peer := <-ctr.peerReady:
			peer.Stop()

		// whenever there is an expired peer to be removed, process it
		case peer := <-ctr.peerStopped:
			_, ok := ctr.peerIndex[peer.String()]
			if !ok {
				return
			}

			delete(ctr.peerIndex, peer.String())
		}
	}
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (ctr *Manager) handleListener(listener *net.TCPListener) {
	// let the waitgroup know when we are done
	defer ctr.wg.Done()

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
			peer.SetManager(ctr),
			peer.SetNetwork(ctr.network),
			peer.SetVersion(ctr.version),
			peer.SetNonce(ctr.nonce),
			peer.SetConnection(conn),
		)
		if err != nil {
			continue
		}

		_ = p
	}
}

// addPeer will try to connect to a new peer and start it on success.
func (ctr *Manager) addPeer() {

	tries := 0
	for {
		// if we tried too many times, give up for this time
		tries++
		if tries > 128 {
			return
		}

		// try to get the best address from the repository
		addr := ctr.repo.Retrieve()

		// check if the address in still unused
		_, ok := ctr.peerIndex[addr.String()]
		if ok {
			continue
		}

		// we initialize a new peer which will callback through a channel on success
		p, err := peer.New(
			peer.SetManager(ctr),
			peer.SetNetwork(ctr.network),
			peer.SetVersion(ctr.version),
			peer.SetNonce(ctr.nonce),
			peer.SetAddress(addr),
		)
		if err != nil {
			return
		}

		_ = p

		ctr.repo.Attempted(addr)
		break
	}
}
