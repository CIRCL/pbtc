package all

import (
	"container/list"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type manager struct {
	addrIn chan string
	peerIn chan *peer
	connIn chan net.Conn
	evtIn  chan event

	sigEvt  chan struct{}
	sigAddr chan struct{}
	sigPeer chan struct{}
	sigConn chan struct{}
	sigInfo chan struct{}

	tickPeer *time.Ticker
	tickInfo *time.Ticker

	peerList  map[string]*peer
	peerQueue *list.List

	waitGroup *sync.WaitGroup
	state     uint32

	network  wire.BitcoinNet
	version  uint32
	numConns uint32
}

func NewManager() *manager {
	mgr := &manager{
		addrIn: make(chan string, bufferManagerAddress),
		peerIn: make(chan *peer, bufferManagerPeer),
		connIn: make(chan net.Conn, bufferManagerConnection),
		evtIn:  make(chan event, bufferManagerEvent),

		tickPeer: time.NewTicker(time.Second / maxConnsPerSec),
		tickInfo: time.NewTicker(logInfoTick),

		peerList:  make(map[string]*peer),
		peerQueue: list.New(),

		waitGroup: &sync.WaitGroup{},
		state:     stateIdle,
	}

	return mgr
}

func (mgr *manager) GetAddrIn() chan<- string {
	return mgr.addrIn
}

func (mgr *manager) GetConnIn() chan<- net.Conn {
	return mgr.connIn
}

func (mgr *manager) Start(network wire.BitcoinNet, version uint32) {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateIdle, stateRunning) {
		return
	}

	log.Println("[MGR] Starting")

	mgr.sigEvt = make(chan struct{}, 1)
	mgr.sigAddr = make(chan struct{}, 1)
	mgr.sigPeer = make(chan struct{}, 1)
	mgr.sigConn = make(chan struct{}, 1)
	mgr.sigInfo = make(chan struct{}, 1)

	mgr.network = network
	mgr.version = version

	mgr.handleAddresses()
	mgr.handlePeers()
	mgr.handleConnections()
	mgr.handleEvents()
	mgr.handlePrinting()

	log.Println("[MGR] Started")
}

func (mgr *manager) Stop() {
	if !atomic.CompareAndSwapUint32(&mgr.state, stateRunning, stateIdle) {
		return
	}

	log.Println("[MGR] Stopping")

	close(mgr.sigConn)
	close(mgr.sigPeer)

	mgr.tickPeer.Stop()
	mgr.peerQueue.Init()

	for _, peer := range mgr.peerList {
		peer.Close()
	}

	log.Println("All peers stopped")

	close(mgr.sigEvt)
	close(mgr.sigAddr)
	close(mgr.sigInfo)

	mgr.tickInfo.Stop()
	mgr.waitGroup.Wait()

	log.Println("[MGR] Stopped")
}

func (mgr *manager) handleAddresses() {
	mgr.waitGroup.Add(1)

	go func() {
		defer mgr.waitGroup.Done()

	AddrLoop:
		for {
			select {
			case _, ok := <-mgr.sigAddr:
				if !ok {
					break AddrLoop
				}

			case addr, ok := <-mgr.addrIn:
				if !ok {
					break AddrLoop
				}

				_, ok = mgr.peerList[addr]
				if ok {
					continue AddrLoop
				}

				if len(mgr.peerList) >= maxNodesTotal {
					continue AddrLoop
				}

				peer := NewPeer(addr, mgr.evtIn)
				mgr.peerList[addr] = peer
				mgr.peerIn <- peer

			}
		}
	}()
}

func (mgr *manager) handlePeers() {
	mgr.waitGroup.Add(1)

	go func() {
		defer mgr.waitGroup.Done()

	PeerLoop:
		for {
			select {
			case _, ok := <-mgr.sigPeer:
				if !ok {
					break PeerLoop
				}

			case peer := <-mgr.peerIn:
				mgr.peerQueue.PushBack(peer)

			case <-mgr.tickPeer.C:
				if mgr.numConns >= maxConnsTotal {
					continue PeerLoop
				}

				element := mgr.peerQueue.Front()
				if element == nil {
					continue PeerLoop
				}

				mgr.peerQueue.Remove(element)
				peer, ok := element.Value.(*peer)
				if !ok {
					continue PeerLoop
				}

				mgr.numConns++
				peer.Connect()
			}
		}
	}()
}

func (mgr *manager) handleConnections() {
	mgr.waitGroup.Add(1)

	go func() {
		defer mgr.waitGroup.Done()

	ConnLoop:
		for {
			select {
			case _, ok := <-mgr.sigConn:
				if !ok {
					break ConnLoop
				}

			case conn, ok := <-mgr.connIn:
				if !ok {
					break ConnLoop
				}

				addr := conn.RemoteAddr().String()
				_, ok = mgr.peerList[addr]
				if ok {
					conn.Close()
					continue ConnLoop
				}

				log.Println("Accepting incoming connection")
				peer := NewPeer(addr, mgr.evtIn)
				mgr.peerList[addr] = peer
				peer.Connection(conn)
				peer.WaitHandshake(mgr.network, mgr.version)
			}
		}
	}()
}

func (mgr *manager) handleEvents() {
	mgr.waitGroup.Add(1)

	go func() {
		defer mgr.waitGroup.Done()

	EvtLoop:
		for {
			select {
			case _, ok := <-mgr.sigEvt:
				if !ok {
					break EvtLoop
				}

			case event, ok := <-mgr.evtIn:
				if !ok {
					break EvtLoop
				}

				switch e := event.(type) {
				case *eventState:
					mgr.processStateChange(e)

				case *eventAddress:
					for _, addr := range e.list {
						mgr.addrIn <- addr
					}
				}
			}
		}
	}()
}

func (mgr *manager) processStateChange(evt *eventState) {
	if atomic.LoadUint32(&mgr.state) != stateRunning {
		return
	}

	switch evt.state {
	case stateIdle:
		mgr.numConns--
		evt.peer.Retry(mgr.peerIn)

	case statePending:
		evt.peer.InitHandshake(mgr.network, mgr.version)

	case stateReady:
		evt.peer.Start()

	case stateRunning:
		evt.peer.Ping()
		evt.peer.Poll()
	}
}

func (mgr *manager) handlePrinting() {
	mgr.waitGroup.Add(1)

	go func() {
		defer mgr.waitGroup.Done()

	InfoLoop:
		for {
			select {
			case _, ok := <-mgr.sigInfo:
				if !ok {
					break InfoLoop
				}

			case <-mgr.tickInfo.C:
				idle := 0
				pending := 0
				ready := 0
				running := 0
				for _, peer := range mgr.peerList {
					switch peer.GetState() {
					case stateIdle:
						idle++
					case statePending:
						pending++
					case stateReady:
						ready++
					case stateRunning:
						running++
					}
				}

				log.Println("[MGR] Peers - Total:", len(mgr.peerList), "Idle:", idle-mgr.peerQueue.Len(), "Queued:", mgr.peerQueue.Len(),
					"Pending:", pending, "Connected:", ready+running)
			}
		}
	}()
}
