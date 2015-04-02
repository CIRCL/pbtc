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

	sigAddr chan struct{}
	sigPeer chan struct{}
	sigConn chan struct{}
	sigInfo chan struct{}

	tickPeer *time.Ticker
	tickInfo *time.Ticker

	peerList  map[string]*peer
	peerQueue *list.List

	waitGroup *sync.WaitGroup

	network  wire.BitcoinNet
	version  uint32
	numConns uint32
	shutdown uint32
}

func NewManager() *manager {
	mgr := &manager{
		addrIn: make(chan string, bufferManagerAddress),
		peerIn: make(chan *peer, bufferManagerPeer),
		connIn: make(chan net.Conn, bufferManagerConnection),
		evtIn:  make(chan event, bufferManagerEvent),

		sigAddr: make(chan struct{}, 1),
		sigPeer: make(chan struct{}, 1),
		sigConn: make(chan struct{}, 1),
		sigInfo: make(chan struct{}, 1),

		tickPeer: time.NewTicker(time.Second / maxConnsPerSec),
		tickInfo: time.NewTicker(logInfoTick),

		peerList:  make(map[string]*peer),
		peerQueue: list.New(),

		waitGroup: &sync.WaitGroup{},
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
	mgr.network = network
	mgr.version = version

	mgr.waitGroup.Add(5)
	go mgr.handleAddresses()
	go mgr.handlePeers()
	go mgr.handleConnections()
	go mgr.handleEvents()
	go mgr.printStatus()
}

func (mgr *manager) Stop() {
	if !atomic.CompareAndSwapUint32(&mgr.shutdown, 0, 1) {
		return
	}

	for _, peer := range mgr.peerList {
		peer.Stop()
	}

	close(mgr.sigAddr)
	close(mgr.sigConn)
	mgr.tickPeer.Stop()
	mgr.tickInfo.Stop()

	mgr.waitGroup.Wait()
}

func (mgr *manager) handleAddresses() {
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

	mgr.waitGroup.Done()
}

func (mgr *manager) handlePeers() {
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

	mgr.peerQueue.Init()
	mgr.waitGroup.Done()
}

func (mgr *manager) handleConnections() {
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

	mgr.waitGroup.Done()
}

func (mgr *manager) handleEvents() {
	for event := range mgr.evtIn {
		switch e := event.(type) {
		case *eventState:
			mgr.handleStateChange(e)

		case *eventAddress:
			for _, addr := range e.list {
				mgr.addrIn <- addr
			}
		}
	}

	mgr.waitGroup.Done()
}

func (mgr *manager) handleStateChange(evt *eventState) {
	if atomic.LoadUint32(&mgr.shutdown) == 1 {
		return
	}

	switch evt.state {
	case stateIdle:
		mgr.numConns--
		evt.peer.Retry(mgr.peerIn)

	case stateConnected:
		evt.peer.InitHandshake(mgr.network, mgr.version)

	case stateReady:
		evt.peer.Start()

	case stateProcessing:
		evt.peer.Ping()
		evt.peer.Poll()
	}
}

func (mgr *manager) printStatus() {
InfoLoop:
	for {
		select {
		case _, ok := <-mgr.sigInfo:
			if !ok {
				break InfoLoop
			}

		case <-mgr.tickInfo.C:
			idle := 0
			connected := 0
			ready := 0
			processing := 0
			for _, peer := range mgr.peerList {
				switch peer.GetState() {
				case stateIdle:
					idle++
				case stateConnected:
					connected++
				case stateReady:
					ready++
				case stateProcessing:
					processing++
				}
			}

			log.Println("Total:", len(mgr.peerList), "Idle:", idle-mgr.peerQueue.Len(), "Queued:", mgr.peerQueue.Len(),
				"Pending:", connected, "Connected:", ready+processing)
		}
	}

	mgr.waitGroup.Done()
}
