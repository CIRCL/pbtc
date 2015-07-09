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

package repository

import (
	"encoding/gob"
	"net"
	"os"
	"sync"
	"time"

	"github.com/CIRCL/pbtc/adaptor"
)

// Repository is the default implementation of the repository interface of the
// Manager module. It creates a simply in-repoory mapping for known nodes and
// regularly save them on the disk.
type Repository struct {
	wg             *sync.WaitGroup
	addrDiscovered chan *net.TCPAddr
	addrAttempted  chan *net.TCPAddr
	addrConnected  chan *net.TCPAddr
	addrSucceeded  chan *net.TCPAddr
	addrRetrieve   chan chan<- *net.TCPAddr
	sigAddr        chan struct{}
	sigRetrieval   chan struct{}
	tickerBackup   *time.Ticker
	tickerPoll     *time.Ticker
	nodeIndex      map[string]*node
	file           *os.File

	log adaptor.Log

	seedsList  []string
	seedsPort  uint16
	backupPath string
	backupRate time.Duration
	nodeLimit  uint32

	invalidRange []*ipRange
}

// New creates a new repository initialized with default values. A variable list
// of options can be provided to override default behaviour.
func New(options ...func(repo *Repository)) (*Repository, error) {
	repo := &Repository{
		wg:             &sync.WaitGroup{},
		nodeIndex:      make(map[string]*node),
		addrDiscovered: make(chan *net.TCPAddr, 1),
		addrAttempted:  make(chan *net.TCPAddr, 1),
		addrConnected:  make(chan *net.TCPAddr, 1),
		addrSucceeded:  make(chan *net.TCPAddr, 1),
		addrRetrieve:   make(chan chan<- *net.TCPAddr, 1),
		sigAddr:        make(chan struct{}),
		sigRetrieval:   make(chan struct{}),
		tickerPoll:     time.NewTicker(30 * time.Minute),

		seedsList:  []string{"testnet-seed.bitcoin.petertodd.org"},
		seedsPort:  18333,
		backupRate: 90 * time.Second,
		backupPath: "nodes.dat",
		nodeLimit:  100000,

		invalidRange: make([]*ipRange, 0, 16),
	}

	for _, option := range options {
		option(repo)
	}

	file, err := os.Create(repo.backupPath)
	if err != nil {
		return nil, err
	}
	repo.file = file

	repo.addRange(newIPRange("0.0.0.0", "0.255.255.255"))       // RFC1700
	repo.addRange(newIPRange("10.0.0.0", "10.255.255.255"))     // RFC1918
	repo.addRange(newIPRange("100.64.0.0", "100.127.255.255"))  // RFC6598
	repo.addRange(newIPRange("127.0.0.0", "127.255.255.255"))   // RFC990
	repo.addRange(newIPRange("169.254.0.0", "169.254.255.255")) // RFC3927
	repo.addRange(newIPRange("172.16.0.0", "172.32.255.255"))   // RFC1918
	repo.addRange(newIPRange("192.0.0.0", "192.0.0.255"))       // RFC5736
	repo.addRange(newIPRange("192.0.2.0", "192.0.2.255"))       // RFC5737
	repo.addRange(newIPRange("192.88.99.0", "192.88.99.255"))   // RFC3068
	repo.addRange(newIPRange("192.168.0.0", "192.168.255.255")) // RFC1918
	repo.addRange(newIPRange("198.18.0.0", "198.19.255.255"))   // RFC2544
	repo.addRange(newIPRange("198.51.100.0", "198.51.100.255")) // RFC5737
	repo.addRange(newIPRange("203.0.113.0", "203.0.113.255"))   // RFC5737
	repo.addRange(newIPRange("224.0.0.0", "239.255.255.255"))   // RFC5771
	repo.addRange(newIPRange("240.0.0.0", "255.255.255.255"))   // RFC6890

	return repo, nil
}

// SetSeeds provides a list of DNS seeds to be used in case of bootstrapping.
func SetSeedsList(seeds ...string) func(*Repository) {
	return func(repo *Repository) {
		repo.seedsList = seeds
	}
}

// SetDefaultPort sets the default port to be used for addresses discovered
// through DNS seeds.
func SetSeedsPort(port uint16) func(*Repository) {
	return func(repo *Repository) {
		repo.seedsPort = port
	}
}

// SetBackupPath sets the path for saving current address & node information.
func SetBackupPath(path string) func(*Repository) {
	return func(repo *Repository) {
		repo.backupPath = path
	}
}

func SetBackupRate(rate time.Duration) func(*Repository) {
	return func(repo *Repository) {
		repo.backupRate = rate
	}
}

func SetNodeLimit(limit uint32) func(*Repository) {
	return func(repo *Repository) {
		repo.nodeLimit = limit
	}
}

func (repo *Repository) Start() {
	repo.log.Info("[REPO] Start: begin")

	repo.tickerBackup = time.NewTicker(repo.backupRate)

	repo.wg.Add(2)
	go repo.goRetrieval()
	go repo.goAddresses()

	repo.bootstrap()

	repo.log.Info("[REPO] Start: complete")
}

// Stop will end all sub-routines and return on clean exit.
func (repo *Repository) Stop() {
	close(repo.sigRetrieval)
	close(repo.sigAddr)

	repo.wg.Wait()

	repo.save()
}

func (repo *Repository) SetLog(log adaptor.Log) {
	repo.log = log
}

// Discovered will submit an address that has been discovered on the Bitcoin
// network.
func (repo *Repository) Discovered(addr *net.TCPAddr) {
	repo.log.Debug("[REPO] Discovered: %v", addr)

	repo.addrDiscovered <- addr
}

// Attempted will mark an address as having been attempted for connection.
func (repo *Repository) Attempted(addr *net.TCPAddr) {
	repo.log.Debug("[REPO] Attempted: %v", addr)

	repo.addrAttempted <- addr
}

// Connected will mark an address as having been used successfully for a TCP
// connection.
func (repo *Repository) Connected(addr *net.TCPAddr) {
	repo.log.Debug("[REPO] Connected: %v", addr)

	repo.addrConnected <- addr
}

// Succeeded will mark an address as having completed the Bitcoin protocol
// handshake successfully.
func (repo *Repository) Succeeded(addr *net.TCPAddr) {
	repo.log.Debug("[REPO] Succeeded: %v", addr)

	repo.addrSucceeded <- addr
}

// Retrieve will send a good candidate address for connecting on the given
// channel.
func (repo *Repository) Retrieve(c chan<- *net.TCPAddr) {
	repo.log.Debug("[REPO] Retrieve: requested")

	repo.addrRetrieve <- c
}

// bootstrap will use a number of dns seeds to discover nodes.
func (repo *Repository) bootstrap() {
	repo.log.Info("[REPO] Bootstrap: getting IPs from %v seeds",
		len(repo.seedsList))

	// iterate over the seeds and try to get the ips
	for _, seed := range repo.seedsList {
		// check if we can look up the ip addresses
		ips, err := net.LookupIP(seed)
		if err != nil {
			continue
		}

		repo.log.Debug("[REPO] Bootstrap: found %v IPs from %v", len(ips), seed)

		// range over the ips and add them to the repository
		for _, ip := range ips {
			addr := &net.TCPAddr{IP: ip, Port: int(repo.seedsPort)}
			repo.Discovered(addr)
		}
	}
}

// save will try to save all current nodes to a file on disk.
func (repo *Repository) save() {
	// create the file, truncating if it already exists
	if repo.file == nil {
		return
	}

	//
	err := repo.file.Truncate(0)
	if err != nil {
		repo.log.Error("failed to truncate repo backup")
		return
	}

	_, err = repo.file.Seek(0, 0)
	if err != nil {
		repo.log.Error("failed to reset repo.file pointer")
		return
	}

	// encode the entire index using gob outputting into repo.file
	enc := gob.NewEncoder(repo.file)
	err = enc.Encode(repo.nodeIndex)
	if err != nil {
		repo.log.Error("failed to encode repo backup")
		return
	}
}

// restore will try to load the previously saved node file.
func (repo *Repository) restore() {
	if repo.file == nil {
		return
	}

	_, err := repo.file.Seek(0, 0)
	if err != nil {
		return
	}

	// decode the entire index using gob reading from the file
	dec := gob.NewDecoder(repo.file)
	err = dec.Decode(&repo.nodeIndex)
	if err != nil {
		return
	}
}

func (repo *Repository) addRange(ipRange *ipRange) {
	repo.invalidRange = append(repo.invalidRange, ipRange)
}

func (repo *Repository) goRetrieval() {
	defer repo.wg.Done()

retrievalLoop:
	for {
		select {
		case _, ok := <-repo.sigRetrieval:
			if !ok {
				break retrievalLoop
			}

		case c := <-repo.addrRetrieve:
			for _, node := range repo.nodeIndex {
				if node.numAttempts >= 1 {
					continue
				}

				if node.lastAttempted.Add(time.Minute * 5).After(time.Now()) {
					continue
				}

				if node.lastConnected.Before(node.lastSucceeded) {
					continue
				}

				if node.lastSucceeded.Add(time.Minute * 15).After(time.Now()) {
					continue
				}

				repo.log.Debug("[REPO] %v retrieved", node)
				c <- node.addr
				continue retrievalLoop
			}
		}
	}
}

func (repo *Repository) goAddresses() {
	defer repo.wg.Done()

	repo.log.Info("[REPO] Address routine started")

addrLoop:
	for {
		select {
		case _, ok := <-repo.sigAddr:
			if !ok {
				break addrLoop
			}

		case <-repo.tickerBackup.C:
			repo.log.Info("[REPO] Saving node index")
			go repo.save()

		case <-repo.tickerPoll.C:
			repo.log.Info("[REPO] Polling DNS seeds")
			go repo.bootstrap()

		case addr := <-repo.addrDiscovered:
			n, ok := repo.nodeIndex[addr.String()]
			if ok {
				n.numSeen++
				continue
			}

			if uint32(len(repo.nodeIndex)) >= repo.nodeLimit {
				return
			}

			ip := addr.IP.To4()
			if ip != nil {
				for _, ipRange := range repo.invalidRange {
					if ipRange.includes(ip) {
						continue addrLoop
					}
				}
			}

			repo.log.Debug("[REPO] %v discovered", addr)
			n = newNode(addr)
			repo.nodeIndex[addr.String()] = n

		case addr := <-repo.addrAttempted:
			n, ok := repo.nodeIndex[addr.String()]
			if !ok {
				repo.log.Warning("[REPO] %v attempted unknown", addr)
				continue
			}

			repo.log.Debug("[REPO] %v attempted", addr)
			n.numAttempts++
			n.lastAttempted = time.Now()

		case addr := <-repo.addrConnected:
			n, ok := repo.nodeIndex[addr.String()]
			if !ok {
				repo.log.Warning("[REPO] %v connected unknown", addr)
				continue
			}

			repo.log.Debug("[REPO] %v connected", addr)
			n.lastConnected = time.Now()

		case addr := <-repo.addrSucceeded:
			n, ok := repo.nodeIndex[addr.String()]
			if !ok {
				repo.log.Warning("[REPO] %v succeeded unknown", addr)
				continue
			}

			repo.log.Debug("[REPO] %v succeeded", addr)
			n.numAttempts = 0
			n.lastSucceeded = time.Now()
		}
	}

	repo.log.Info("[REPO] Address routine stopped")
}
