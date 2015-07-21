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
)

// Manager is the module responsible for peer management. It will initialize
// new incoming & outgoing peers and take care of state transitions. As the
// main control instance, it defines most of the behaviour of our peer.
type Manager struct {
	wg  *sync.WaitGroup
	sig chan struct{}

	incomingQ  chan adaptor.Peer
	outgoingQ  chan adaptor.Peer
	connectedQ chan adaptor.Peer
	readyQ     chan adaptor.Peer
	stoppedQ   chan adaptor.Peer

	tickerT *time.Ticker

	peerIndex   *parmap.ParMap
	listenIndex map[string]*net.TCPListener

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
		wg:  &sync.WaitGroup{},
		sig: make(chan struct{}),

		incomingQ:  make(chan adaptor.Peer, 1),
		outgoingQ:  make(chan adaptor.Peer, 1),
		connectedQ: make(chan adaptor.Peer, 1),
		readyQ:     make(chan adaptor.Peer, 1),
		stoppedQ:   make(chan adaptor.Peer, 1),

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

	mgr.tickerT = time.NewTicker(mgr.tickerInterval)

	mgr.wg.Add(2)
	go mgr.goTicker()
	go mgr.goEvents()
	go mgr.goPeers()

	mgr.log.Info("[MGR] Start: completed")
}

// Close will clean-up before shutdown.
func (mgr *Manager) Stop() {
	mgr.log.Info("[MGR] Stop: begin")

	close(mgr.sig)

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

func (mgr *Manager) Outgoing(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Outgoing: %v", p)

	mgr.outgoingQ <- p
}

func (mgr *Manager) Incoming(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Incoming: %v", p)

	mgr.incomingQ <- p
}

// Connected signals to the manager that we have successfully established a
// TCP connection to a peer.
func (mgr *Manager) Connected(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Connected: %v", p)

	mgr.connectedQ <- p
}

// Ready signals to the manager that we have successfully completed the Bitcoin
// protocol handshake with a peer.
func (mgr *Manager) Ready(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Ready: %v", p)

	mgr.readyQ <- p
}

// Stopped signals to the manager that the connection to this peer has been
// shut down.
func (mgr *Manager) Stopped(p adaptor.Peer) {
	mgr.log.Debug("[MGR] Stopped: %v", p)

	mgr.stoppedQ <- p
}

func (mgr Manager) goTicker() {
	defer mgr.wg.Done()

TickerLoop:
	for {
		select {
		case _, ok := <-mgr.sig:
			if !ok {
				break TickerLoop
			}

		// print manager information to the log
		case <-mgr.tickerT.C:
			mgr.log.Info("[MGR] %v total peers managed", mgr.peerIndex.Count())
		}
	}
}

func (mgr *Manager) goEvents() {
PeerLoop:
	for {
		select {
		case _, ok := <-mgr.sig:
			if !ok {
				break PeerLoop
			}

		// manage peers that have successfully connected
		case p := <-mgr.connectedQ:
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
		case p := <-mgr.readyQ:
			if !mgr.peerIndex.Has(p) {
				mgr.log.Warning("[MGR] %v already ready", p)
				p.Stop()
				continue
			}

			mgr.log.Debug("[MGR] %v ready", p)
			mgr.repo.Succeeded(p.Addr())
			p.Poll()

		// manage peers that have dropped the connection
		case p := <-mgr.stoppedQ:
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
		case p := <-mgr.connectedQ:
			p.Stop()
			break

		case p := <-mgr.readyQ:
			p.Stop()
			break

		case p := <-mgr.stoppedQ:
			mgr.peerIndex.Remove(p)
			break
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
		case _, ok := <-mgr.sig:
			if !ok {
				break PeerLoop
			}

		// manage peers that have successfully connected
		case _ = <-mgr.incomingQ:

		case _ = <-mgr.outgoingQ:

		}
	}
}
