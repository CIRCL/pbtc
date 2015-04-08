package all

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	nodeRepo    *repository
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

// NewManager creates a new manager with all necessary initializations done.
func NewManager() *manager {
	mgr := &manager{
		peerIndex: make(map[string]*peer),
		sigPeer:   make(chan struct{}, 1),
		sigConn:   make(chan struct{}, 1),
		peerNew:   make(chan *peer, bufferManagerNew),
		peerDone:  make(chan *peer, bufferManagerDone),

		connTicker: time.NewTicker(time.Second / maxConnsPerSec),

		wg:    &sync.WaitGroup{},
		state: stateIdle,
	}

	return mgr
}

// Start starts the manager, with run-time options passed in as parameters.
func (mgr *manager) Start(repo *repository, network wire.BitcoinNet, version uint32) {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateIdle, stateBusy) {
		return
	}

	mgr.nodeRepo = repo
	mgr.network = network
	mgr.version = version

	mgr.wg.Add(3)
	go mgr.handleListeners()
	go mgr.handleConnections()
	go mgr.handlePeers()

	atomic.StoreUint32(&mgr.state, stateRunning)
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *manager) Stop() {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateRunning, stateBusy) {
		return
	}

	for _, peer := range mgr.peerIndex {
		peer.Stop()
	}

	close(mgr.sigConn)

	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	close(mgr.sigPeer)

	mgr.wg.Wait()

	atomic.StoreUint32(&mgr.state, stateIdle)
}

func (mgr *manager) handleListeners() {
	defer mgr.wg.Done()

	ips := FindLocalIPs()
	for _, ip := range ips {
		addr, err := net.ResolveTCPAddr("tcp", ip.String())
		if err != nil {
			continue
		}

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
		go mgr.processListener(listener)
	}
}

func (mgr *manager) handleConnections() {
	defer mgr.wg.Done()

ConnLoop:
	for {
		select {
		case _, ok := <-mgr.sigConn:
			if !ok {
				break ConnLoop
			}

		case <-mgr.connTicker.C:
			if len(mgr.peerIndex) <= maxPeerCount {
				mgr.addPeer()
			}
		}
	}
}

func (mgr *manager) handlePeers() {
	defer mgr.wg.Done()

PeerLoop:
	for {
		select {
		case _, ok := <-mgr.sigPeer:
			if !ok {
				break PeerLoop
			}

		case peer := <-mgr.peerNew:
			mgr.processNewPeer(peer)

		case peer := <-mgr.peerDone:
			mgr.processDonePeer(peer)
		}
	}
}

func (mgr *manager) addPeer() {
	tries := 0
	var addr *net.TCPAddr
	for {
		addr, err := mgr.nodeRepo.Get()
		if err != nil {
			return
		}

		tries++
		if tries > 128 {
			return
		}

		_, ok := mgr.peerIndex[addr.String()]
		if !ok {
			continue
		}
	}

	peer, err := NewOutgoingPeer(mgr, addr, mgr.network, mgr.version, mgr.nonce)
	if err != nil {
		return
	}

	mgr.peerNew <- peer
}

func (mgr *manager) processNewPeer(peer *peer) {
	mgr.peerIndex[peer.String()] = peer
}

func (mgr *manager) processDonePeer(peer *peer) {
	_, ok := mgr.peerIndex[peer.String()]
	if !ok {
		return
	}

	delete(mgr.peerIndex, peer.String())
}

func (mgr *manager) processListener(listener *net.TCPListener) {
	defer mgr.wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		peer, err := NewIncomingPeer(mgr, conn, mgr.network, mgr.version, mgr.nonce)
		if err != nil {
			continue
		}

		mgr.peerNew <- peer
	}
}
