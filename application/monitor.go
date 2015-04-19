package application

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/domain"
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
type Monitor struct {
	repo        domain.Repository
	peerIndex   map[string]*domain.Peer
	listenIndex map[string]*net.TCPListener
	sigPeer     chan struct{}
	sigConn     chan struct{}
	peerNew     chan *domain.Peer
	peerDone    chan *domain.Peer
	connTicker  *time.Ticker
	wg          *sync.WaitGroup
	state       uint32
	network     wire.BitcoinNet
	version     uint32
	nonce       uint64
}

// NewManager returns a new manager with all necessary variables initialized.
func NewMonitor(options ...func(mon *Monitor)) *Monitor {
	mon := &Monitor{
		peerIndex:   make(map[string]*domain.Peer),
		listenIndex: make(map[string]*net.TCPListener),
		sigPeer:     make(chan struct{}, 1),
		sigConn:     make(chan struct{}, 1),
		peerNew:     make(chan *domain.Peer, 1),
		peerDone:    make(chan *domain.Peer, 1),
		connTicker:  time.NewTicker(time.Second / 4),
		wg:          &sync.WaitGroup{},
		state:       stateIdle,
	}

	for _, option := range options {
		option(mon)
	}

	if mon.network == 0 {
		mon.network = wire.TestNet3
	}

	if mon.version == 0 {
		mon.version = wire.RejectVersion
	}

	return mon
}

func (mon *Monitor) Connected(peer *domain.Peer) {
}

func (mon *Monitor) Started(peer *domain.Peer) {

}

func (mon *Monitor) Stopped(peer *domain.Peer) {

}

// Start starts the manager, with run-time options passed in as parameters. This allows
// us to stop and restart the manager with a different protocol version, network or even
// repository of nodes.
func (mon *Monitor) Start(network wire.BitcoinNet, version uint32) {
	// we can only start the manager if it is in idle state and ready to be started
	if !atomic.CompareAndSwapUint32(&mon.state, stateIdle, stateBusy) {
		return
	}

	log := logging.MustGetLogger("pbtc")
	log.Info("Manager starting up...")

	// set the parameters for the nodes and connections we will create
	mon.network = network
	mon.version = version

	// listen on local IPs for incoming peers
	mon.createListeners()

	// here, we start all handlers that execute concurrently
	// we add them to the waitgrop so that we can cleanly shutdown later
	mon.wg.Add(2)
	go mon.handleConnections()
	go mon.handlePeers()

	// at this point, start-up is complete and we can set the new state
	atomic.StoreUint32(&mon.state, stateRunning)
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mon *Monitor) Stop() {
	// we can only stop the manager if we are currently in running state
	if !atomic.CompareAndSwapUint32(&mon.state, stateRunning, stateBusy) {
		return
	}

	log := logging.MustGetLogger("pbtc")

	// first we will stop every peer - this is a blocking operation
	log.Debug("Stopping peers")
	for _, peer := range mon.peerIndex {
		peer.Stop()
	}

	// here, we close the channel to signal the connection handler to stop
	close(mon.sigConn)

	// the listener handler already quits after launching all listeners
	// we thus only need to close all listeners and wait for their routines to stop
	for _, listener := range mon.listenIndex {
		listener.Close()
	}

	// finally, we signal the peer listener to stop processing as well
	close(mon.sigPeer)

	// we then wait for all handlers to finish cleanly
	log.Debug("Waiting for handlers to quit")
	mon.wg.Wait()

	// at this point, all handlers have stopped and we are back in idle state
	log.Info("Manager shutdown complete")
	atomic.StoreUint32(&mon.state, stateIdle)
}

// createListeners tries to start a listener on every local IP to accept
// connections. It should be called as a go routine.
func (mon *Monitor) createListeners() {
	log := logging.MustGetLogger("pbtc")

	// get all IPs on local interfaces and iterate through them
	ips, err := domain.FindLocalIPs()
	if err != nil {
		log.Error("Could not find local IPs")
		return
	}
	log.Info("%v local IP(s) found", len(ips))

	for _, ip := range ips {
		// if we can't convert into a TCP address, skip
		addr, err := net.ResolveTCPAddr("tcp", ip.String()+":"+strconv.Itoa(18333))
		if err != nil {
			log.Warning("%v: could not convert to TCP address", ip)
			continue
		}

		// if we are already listening on this address, skip
		_, ok := mon.listenIndex[addr.String()]
		if ok {
			log.Notice("%v: already in listener index", addr)
			continue
		}

		// if we can't create the listener, skip
		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			log.Warning("%v: could not create TCP listener", addr)
			continue
		}

		// add the listener to our index and start an accepting handler
		// we again need to add it to the waitgroup if we want to exit cleanly
		log.Info("%v: listening for connections", addr)
		mon.listenIndex[addr.String()] = listener
		mon.wg.Add(1)
		go mon.handleListener(listener)
	}
}

// handleConnections attempts to establish new connections at the configured
// rate as long as we are not at the maximum number of connections.
func (mon *Monitor) handleConnections() {
	log := logging.MustGetLogger("pbtc")
	log.Debug("Connection handler started")

	// let the waitgroup know when we are done
	defer mon.wg.Done()

ConnLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mon.sigConn:
			if !ok {
				break ConnLoop
			}

		// the ticker will signal each time we can attempt a new connection
		// if we don't have too many peers yet, try to create a new one
		case <-mon.connTicker.C:
			if len(mon.peerIndex) < 128 {
				mon.addPeer()
			}
		}
	}

	log.Debug("Connection handler stopped")
}

// handlePeers will execute householding operations on new peers and peers
// that have expired. It should be used to keep track of peers and to convey
// application state to the peers.
func (mon *Monitor) handlePeers() {
	log := logging.MustGetLogger("pbtc")
	log.Debug("Peer handler started")

	// let the waitgroup know when we are done
	defer mon.wg.Done()

PeerLoop:
	for {
		select {
		// this is the signal to quit, so break the outer loop
		case _, ok := <-mon.sigPeer:
			if !ok {
				break PeerLoop
			}

		// whenever there is a new peer to be added, process it
		case peer := <-mon.peerNew:
			mon.processNewPeer(peer)

		// whenever there is an expired peer to be removed, process it
		case peer := <-mon.peerDone:
			mon.processDonePeer(peer)
		}
	}

	log.Debug("Peer handler stopped")
}

// processListener is a dedicated loop to be run for every local IP that we
// want to listen on. It should be run as a go routine and will try accepting
// new connections.
func (mon *Monitor) handleListener(listener *net.TCPListener) {
	log := logging.MustGetLogger("pbtc")
	log.Debug("%v: listener handler started", listener.Addr())

	// let the waitgroup know when we are done
	defer mon.wg.Done()

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
			log.Warning("%v: could not accept connection (%v)", listener.Addr(), err)
			break
		}

		// create a new incoming peer for the given connection
		// if the connection is valid, the peer will notify the manager on its own
		p, err := domain.NewPeer(
			domain.SetManager(mon),
			domain.SetNetwork(mon.network),
			domain.SetVersion(mon.version),
			domain.SetNonce(mon.nonce),
			domain.SetConnection(conn),
		)
		if err != nil {
			log.Error("%v: could not create incoming peer (%v)", conn.RemoteAddr(), err)
			continue
		}

		_ = p
	}

	log.Debug("%v: listener handler stopped", listener.Addr())
}

// addPeer will try to connect to a new peer and start it on success.
func (mon *Monitor) addPeer() {
	log := logging.MustGetLogger("pbtc")

	tries := 0
	for {
		// if we tried too many times, give up for this time
		tries++
		if tries > 128 {
			log.Notice("Couldn't get good address")
			return
		}

		// try to get the best address from the repository
		addr := mon.repo.Get()

		// check if the address in still unused
		_, ok := mon.peerIndex[addr.String()]
		if ok {
			log.Notice("%v: already connected", addr)
			continue
		}

		// we initialize a new peer which will callback through a channel on success
		p, err := domain.NewPeer(
			domain.SetManager(mon),
			domain.SetNetwork(mon.network),
			domain.SetVersion(mon.version),
			domain.SetNonce(mon.nonce),
			domain.SetAddress(addr),
		)
		if err != nil {
			log.Error("%v: couldn't create peer (%v)", addr, err)
			return
		}

		_ = p

		log.Info("%v: new peer initialized", addr)
		mon.repo.Attempted(addr)
		break
	}
}

// processNewPeer is what we do with new initialized peers that are added to
// the manager. The peers should be in a connected state so we can start them
// and add them to our index.
func (mon *Monitor) processNewPeer(peer *domain.Peer) {
	log := logging.MustGetLogger("pbtc")
	mon.repo.Connected(peer.Addr())
	mon.repo.Succeeded(peer.Addr())

	_, ok := mon.peerIndex[peer.String()]
	if ok {
		log.Warning("%v: peer already exists", peer)
		return
	}

	if len(mon.peerIndex) >= 128 {
		log.Notice("%v: maximum peer number reached, discarding", peer)
		return
	}

	log.Debug("%v: starting connected peer", peer)
	mon.peerIndex[peer.String()] = peer
}

// processDonePeer is what we do to expired peers. They failed in some way and
// already initialized shutdown on their own, so we just need to remove them
// from our index.
func (mon *Monitor) processDonePeer(peer *domain.Peer) {
	log := logging.MustGetLogger("pbtc")
	_, ok := mon.peerIndex[peer.String()]
	if !ok {
		log.Notice("%v: peer does not exist", peer)
		return
	}

	delete(mon.peerIndex, peer.String())
	log.Debug("%v: done peer removed", peer)
}
