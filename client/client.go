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

package client

import (
	"net"
	"sync"
	"time"

	"github.com/CIRCL/pbtc/adaptor"
)

type Client struct {
	wg  *sync.WaitGroup
	sig chan struct{}

	addrQ chan *net.TCPAddr
	addrT *time.Ticker

	log adaptor.Log
	mgr adaptor.Manager

	connRate time.Duration
}

func NewClient(options ...func(*Client)) (*Client, error) {
	c := &Client{
		wg:       &sync.WaitGroup{},
		sig:      make(chan struct{}),
		addrQ:    make(chan string, 1),
		connRate: time.Second / time.Duration(10),
	}

	for _, option := range options {
		option(c)
	}

	return c, nil
}

func SetConnectionRate(rate time.Duration) func(*Client) {
	return func(c *Client) {
		c.connRate = rate
	}
}

func (c *Client) Start() {
	c.addrT = time.NewTicker(time.Second / c.connRate)

	c.wg.Add(1)
	go c.goConnect()
}

func (c *Client) Stop() {
	close(c.sig)

	c.wg.Wait()
}

func (c *Client) SetLog(log adaptor.Log) {
	c.log = log
}

func (c *Client) SetManager(mgr adaptor.Manager) {
	c.mgr = mgr
}

func (c *Client) goConnect() {
	defer c.wg.Done()

ConnectLoop:
	for {
		select {
		case _, ok := <-c.sig:
			if !ok {
				break ConnectLoop
			}

		case addr := <-c.addrQ:
			<-c.addrTicker

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
		}
	}
}
