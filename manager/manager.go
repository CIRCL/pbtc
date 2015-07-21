// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package manager

import (
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
	"github.com/CIRCL/pbtc/peer"
)

// Manager is the module responsible for peer management. It will initialize
// new incoming & outgoing peers and take care of state transitions. As the
// main control instance, it defines most of the behaviour of our peer.
type Manager struct {
	wg            *sync.WaitGroup
	peerSig       chan struct{}
	addrSig       chan struct{}
	addrQ         chan *net.TCPAddr
	connQ         chan *net.TCPConn
	peerConnected chan adaptor.Peer
	peerReady     chan adaptor.Peer
	peerStopped   chan adaptor.Peer
	peerIndex     *parmap.ParMap
	listenIndex   map[string]*net.TCPListener
	addrTicker    *time.Ticker
	infoTicker    *time.Ticker

	network        wire.BitcoinNet
	version        uint32
	connRate       time.Duration
	tickerInterval time.Duration
	connLimit      int

	log  adaptor.Log
	repo adaptor.Repository
	tkr  adaptor.Tracker
	pro  []adaptor.Processor

	nonce uint64
}

// New returns a new manager initialized with the given options.
func New(options ...func(mgr *Manager)) (*Manager, error) {
	mgr := &Manager{
		wg: &sync.WaitGroup{},

		peerSig:       make(chan struct{}),
		addrSig:       make(chan struct{}),
		addrQ:         make(chan *net.TCPAddr, 1),
		connQ:         make(chan *net.TCPConn, 1),
		peerConnected: make(chan adaptor.Peer, 1),
		peerReady:     make(chan adaptor.Peer, 1),
		peerStopped:   make(chan adaptor.Peer, 1),

		peerIndex:   parmap.New(),
		listenIndex: make(map[string]*net.TCPListener),

		network:        wire.TestNet3,
		version:        wire.RejectVersion,
		connRate:       time.Second / 10,
		connLimit:      100,
		tickerInterval: time.Second * 10,
	}

	nonce, err := wire.RandomUint64()
	if err != nil {
		return nil, err
	}

	mgr.nonce = nonce

	for _, option := range options {
		option(mgr)
	}

	return mgr, nil
}

// SetNetwork has to be passed as a parameter on manager creation. It sets the
// Bitcoin network to be used (main, test, regression, ...).
func SetProtocolMagic(network wire.BitcoinNet) func(*Manager) {
	return func(mgr *Manager) {
		mgr.network = network
	}
}

// SetVersion has to be passed as a parameter on manager creation. It sets the
// maximum protocol version to be used for peer communication.
func SetProtocolVersion(version uint32) func(*Manager) {
	return func(mgr *Manager) {
		mgr.version = version
	}
}

// SetConnectionRate has to be passed as a parameter on manager creation. It
// sets the maximum number of attempted TCP connections per second.
func SetConnectionRate(connRate time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.connRate = connRate
	}
}

// SetPeerLimit has to be passed as a parameter on manager creation. It sets
// the maximum number of concurrent TCP connections, thus limiting the total
// number of connecting and connected peers.
func SetConnectionLimit(connLimit int) func(*Manager) {
	return func(mgr *Manager) {
		mgr.connLimit = connLimit
	}
}

func SetTickerInterval(tickerInterval time.Duration) func(*Manager) {
	return func(mgr *Manager) {
		mgr.tickerInterval = tickerInterval
	}
}

func (mgr *Manager) Start() {
	mgr.log.Info("[MGR] Start: begin")

	mgr.addrTicker = time.NewTicker(mgr.connRate)
	mgr.infoTicker = time.NewTicker(mgr.tickerInterval)

	mgr.wg.Add(3)
	go mgr.goPeers()
	go mgr.goAddresses()
	go mgr.goTicker()

	mgr.log.Info("[MGR] Start: completed")
}

// Close will clean-up before shutdown.
func (mgr *Manager) Stop() {
	mgr.log.Info("[MGR] Stop: begin")

	close(mgr.addrSig)
	close(mgr.peerSig)

	for s := range mgr.peerIndex.Iter() {
		p := s.(adaptor.Peer)
		p.Stop()
	}

	mgr.wg.Wait()

	mgr.log.Info("[MGR] Stop: completed")
}

func (mgr *Manager) SetLog(log adaptor.Log) {
	mgr.log = log
}

func (mgr *Manager) SetRepository(repo adaptor.Repository) {
	mgr.repo = repo
}

func (mgr *Manager) SetTracker(tkr adaptor.Tracker) {
	mgr.tkr = tkr
}

func (mgr *Manager) AddProcessor(pro adaptor.Processor) {
	mgr.pro = append(mgr.pro, pro)
}

func (mgr *Manager) Connection(conn *net.TCPConn) {
	mgr.log.Debug("[MGR] Connection: %v", conn.RemoteAddr)

	mgr.connQ <- conn
}

// Connected signals to the manager that we have successfully established a
// TCP connection to a peer.
func (mgr *Manager) Connected(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Connected: %v", p)

	mgr.peerConnected <- p
}

// Ready signals to the manager that we have successfully completed the Bitcoin
// protocol handshake with a peer.
func (mgr *Manager) Ready(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Ready: %v", p)

	mgr.peerReady <- p
}

// Stopped signals to the manager that the connection to this peer has been
// shut down.
func (mgr *Manager) Stopped(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Stopped: %v", p)

	mgr.peerStopped <- p
}

// to be called from a go routine
// will request and receive addresses for our connection attempts
func (mgr *Manager) goAddresses() {
	defer mgr.wg.Done()

AddressLoop:
	for {
		select {
		case _, ok := <-mgr.addrSig:
			if !ok {
				break AddressLoop
			}

		// at the pace of addr ticker, we request addresses to connect to
		// as long as we have not reached the peer limit
		case <-mgr.addrTicker.C:
			if mgr.peerIndex.Count() >= mgr.connLimit {
				continue
			}

			mgr.repo.Retrieve(mgr.addrQ)
		}
	}
}

func (mgr Manager) goTicker() {
	defer mgr.wg.Done()

TickerLoop:
	for {
		select {
		case _, ok := <-mgr.peerSig:
			if !ok {
				break TickerLoop
			}

		// print manager information to the log
		case <-mgr.infoTicker.C:
			mgr.log.Info("[MGR] %v total peers managed", mgr.peerIndex.Count())
		}
	}
}

// to be called from a go routine
// will manage all peer connection/disconnection
func (mgr *Manager) goPeers() {
	defer mgr.wg.Done()

PeerLoop:
	for {
		select {
		case _, ok := <-mgr.peerSig:
			if !ok {
				break PeerLoop
			}

		// create new outgoing peers for received addresses
		case addr := <-mgr.addrQ:
			if mgr.peerIndex.HasKey(addr.String()) {
				mgr.log.Debug("[MGR] %v already created", addr)
				continue
			}

			if mgr.peerIndex.Count() >= mgr.connLimit {
				mgr.log.Debug("[MGR] %v discarded, limit reached", addr)
				continue
			}

			p, err := peer.New(
				peer.SetLog(mgr.log),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetNetwork(mgr.network),
				peer.SetVersion(mgr.version),
				peer.SetNonce(mgr.nonce),
				peer.SetAddress(addr),
				peer.SetTracker(mgr.tkr),
				peer.SetProcessors(mgr.pro),
			)
			if err != nil {
				mgr.log.Error("[MGR] %v failed outbound (%v)", addr, err)
				continue
			}

			mgr.log.Debug("[MGR] %v created", p)
			mgr.peerIndex.Insert(p)
			mgr.repo.Attempted(p.Addr())
			p.Connect()

		// create new incoming peer for received connections
		case conn := <-mgr.connQ:
			addr := conn.RemoteAddr()
			if mgr.peerIndex.HasKey(addr.String()) {
				mgr.log.Notice("[MGR] limit reached, %v not accepted", addr)
				conn.Close()
				continue
			}

			if mgr.peerIndex.Count() >= mgr.connLimit {
				mgr.log.Debug("[MGR] %v disconnected, limit reached", addr)
				conn.Close()
				continue
			}

			p, err := peer.New(
				peer.SetLog(mgr.log),
				peer.SetRepository(mgr.repo),
				peer.SetManager(mgr),
				peer.SetNetwork(mgr.network),
				peer.SetVersion(mgr.version),
				peer.SetNonce(mgr.nonce),
				peer.SetConnection(conn),
				peer.SetTracker(mgr.tkr),
				peer.SetProcessors(mgr.pro),
			)
			if err != nil {
				mgr.log.Error("[MGR] %v failed inbound (%v)", addr, err)
				continue
			}

			mgr.log.Debug("[MGR] %v accepted", p)
			mgr.peerIndex.Insert(p)
			mgr.repo.Attempted(p.Addr())
			mgr.repo.Connected(p.Addr())
			p.Start()

		// manage peers that have successfully connected
		case p := <-mgr.peerConnected:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v connected unknown", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v connected", p)
			mgr.repo.Connected(p.Addr())
			p.Start()
			p.Greet()

		// manage peers that have completed the handshake
		case p := <-mgr.peerReady:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v already ready", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v ready", p)
			mgr.repo.Succeeded(p.Addr())
			p.Poll()

		// manage peers that have dropped the connection
		case p := <-mgr.peerStopped:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v done unknown", p)
				continue
			}

			mgr.log.Debug("[MGR] %v: done", p)
			mgr.peerIndex.Remove(p)
		}
	}

	// wait for all peers to stop and drain the channels
	for mgr.peerIndex.Count() > 0 {
		select {
		case <-mgr.addrQ:
			break

		case conn := <-mgr.connQ:
			conn.Close()
			break

		case p := <-mgr.peerConnected:
			p.Stop()
			break

		case p := <-mgr.peerReady:
			p.Stop()
			break

		case p := <-mgr.peerStopped:
			mgr.peerIndex.Remove(p)
			break
		}
	}
}
