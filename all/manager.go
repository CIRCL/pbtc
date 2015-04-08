package all

import (
	"log"
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

	peerNew  chan *peer
	peerDone chan *peer

	connTicker *time.Ticker

	wg    *sync.WaitGroup
	state uint32

	network wire.BitcoinNet
	version uint32
	nonce   uint64
}

func NewManager() *manager {
	mgr := &manager{
		nodeRepo:  NewRepository(),
		peerIndex: make(map[string]*peer),

		peerNew:  make(chan *peer, bufferManagerNew),
		peerDone: make(chan *peer, bufferManagerDone),

		connTicker: time.NewTicker(time.Second / maxConnsPerSec),

		wg:    &sync.WaitGroup{},
		state: stateIdle,
	}

	return mgr
}

func (mgr *manager) GetNetwork() wire.BitcoinNet {
	return mgr.network
}

func (mgr *manager) GetVersion() uint32 {
	return mgr.version
}

func (mgr *manager) GetNonce() uint64 {
	return mgr.nonce
}

// Start starts the peer manager on a certain network and version.
func (mgr *manager) Start(network wire.BitcoinNet, version uint32) {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateIdle, stateRunning) {
		return
	}

	log.Println("[MGR] Starting")

	mgr.network = network
	mgr.version = version

	mgr.wg.Add(2)
	go mgr.handleListeners()
	go mgr.handlePeers()

	log.Println("[MGR] Started")
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *manager) Stop() {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateRunning, stateShutdown) {
		return
	}

	log.Println("[MGR] Stopping")

	for _, listener := range mgr.listenIndex {
		listener.Close()
	}

	mgr.wg.Wait()

	log.Println("[MGR] Stopped")
}

func (mgr *manager) handlePeers() {
	defer mgr.wg.Done()

	for {
		if atomic.LoadUint32(&mgr.state) == stateShutdown {
			break
		}

		select {
		case <-mgr.connTicker.C:
			if len(mgr.peerIndex) <= maxPeerCount {
				addr := mgr.nodeRepo.Get()
				peer, err := NewOutgoingPeer(mgr, addr)
				if err != nil {
					break
				}

				mgr.peerNew <- peer
			}

		default:
			break
		}

		select {
		case peer := <-mgr.peerNew:
			mgr.processNewPeer(peer)

		case peer := <-mgr.peerDone:
			mgr.processDonePeer(peer)
		}
	}
}

func (mgr *manager) handleListeners() {
	defer mgr.wg.Done()

	ips := FindIPs()
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

		peer, err := NewIncomingPeer(mgr, conn)
		if err != nil {
			continue
		}

		mgr.peerNew <- peer
	}
}
