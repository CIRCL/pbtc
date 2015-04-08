package all

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	nodeRepo  *repository
	peerIndex map[string]*peer

	peerNew  chan *peer
	peerDone chan *peer

	connTicker *time.Ticker

	waitGroup *sync.WaitGroup
	state     uint32

	network wire.BitcoinNet
	version uint32
	nonce   uint64
}

func NewManager() *manager {
	mgr := &manager{
		nodeRepo:  NewRepository(),
		peerIndex: make(map[string]*peer),

		peerNew:  make(chan *peer, bufferManagerPeer),
		peerDone: make(chan *peer, bufferManagerPeer),

		connTicker: time.NewTicker(time.Second / maxConnsPerSec),

		waitGroup: &sync.WaitGroup{},
		state:     stateIdle,
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

	mgr.waitGroup.Add(1)

	go mgr.handlePeers()

	log.Println("[MGR] Started")
}

// Stop cleanly shuts down the manager so it can be restarted later.
func (mgr *manager) Stop() {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateRunning, stateShutdown) {
		return
	}

	log.Println("[MGR] Stopping")

	mgr.waitGroup.Wait()

	log.Println("[MGR] Stopped")
}

func (mgr *manager) handlePeers() {
	defer mgr.waitGroup.Done()

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
