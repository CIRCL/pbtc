package repository

import (
	"bytes"
	"encoding/gob"
	"net"
	"os"
	"sync"
	"sync/atomic"
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

	seeds      []string
	backupPath string

	log adaptor.Logger

	done             uint32
	restoreEnabled   bool
	defaultPort      int
	bootstrapEnabled bool
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
		tickerBackup:   time.NewTicker(90 * time.Second),
		tickerPoll:     time.NewTicker(30 * time.Minute),
		defaultPort:    18333,

		seeds:          []string{"testnet-seed.bitcoin.petertodd.org"},
		backupPath:     "nodes.dat",
		restoreEnabled: true,
	}

	for _, option := range options {
		option(repo)
	}

	if repo.restoreEnabled {
		repo.restore()
	}

	if len(repo.nodeIndex) == 0 {
		repo.bootstrapEnabled = true
	}

	repo.start()

	if repo.bootstrapEnabled {
		repo.bootstrap()
	}

	return repo, nil
}

func SetLogger(log adaptor.Logger) func(*Repository) {
	return func(repo *Repository) {
		repo.log = log
	}
}

func SetSeeds(seeds ...string) func(*Repository) {
	return func(repo *Repository) {
		repo.seeds = make([]string, len(seeds))

		for i, seed := range seeds {
			repo.seeds[i] = seed
		}
	}
}

func SetBackupPath(path string) func(*Repository) {
	return func(repo *Repository) {
		repo.backupPath = path
	}
}

func SetDefaultPort(port int) func(*Repository) {
	return func(repo *Repository) {
		repo.defaultPort = port
	}
}

func DisableRestore() func(*Repository) {
	return func(repo *Repository) {
		repo.restoreEnabled = false
	}
}

func (repo *Repository) Stop() {
	if atomic.SwapUint32(&repo.done, 1) == 1 {
		return
	}

	close(repo.sigRetrieval)
	close(repo.sigAddr)

	repo.wg.Wait()

	repo.save()

	repo.log.Info("[REPO] Shutdown complete")
}

func (repo *Repository) Discovered(addr *net.TCPAddr) {
	repo.addrDiscovered <- addr
}

func (repo *Repository) Attempted(addr *net.TCPAddr) {
	repo.addrAttempted <- addr
}

func (repo *Repository) Connected(addr *net.TCPAddr) {
	repo.addrConnected <- addr
}

func (repo *Repository) Succeeded(addr *net.TCPAddr) {
	repo.addrSucceeded <- addr
}

func (repo *Repository) Retrieve(c chan<- *net.TCPAddr) {
	repo.addrRetrieve <- c
}

func (repo *Repository) start() {
	repo.wg.Add(2)
	go repo.goRetrieval()
	go repo.goAddresses()

	repo.log.Info("[REPO] Initialization complete")
}

// bootstrap will use a number of dns seeds to discover nodes.
func (repo *Repository) bootstrap() {
	// iterate over the seeds and try to get the ips
	for _, seed := range repo.seeds {
		// check if we can look up the ip addresses
		ips, err := net.LookupIP(seed)
		if err != nil {
			continue
		}

		// range over the ips and add them to the repository
		for _, ip := range ips {
			addr := &net.TCPAddr{IP: ip, Port: repo.defaultPort}
			repo.Discovered(addr)
		}
	}
}

// save will try to save all current nodes to a file on disk.
func (repo *Repository) save() {
	// create the file, truncating if it already exists
	file, err := os.Create(repo.backupPath)
	if err != nil {
		return
	}
	defer file.Close()

	// encode the entire index using gob outputting into file
	enc := gob.NewEncoder(file)
	err = enc.Encode(repo.nodeIndex)
	if err != nil {
		return
	}
}

// restore will try to load the previously saved node file.
func (repo *Repository) restore() {
	// open the nodes file in read-only mode
	file, err := os.Open(repo.backupPath)
	if err != nil {
		return
	}
	defer file.Close()

	// decode the entire index using gob reading from the file
	dec := gob.NewDecoder(file)
	err = dec.Decode(&repo.nodeIndex)
	if err != nil {
		return
	}
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

				if node.addr.Port != repo.defaultPort {
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
			repo.save()

		case <-repo.tickerPoll.C:
			repo.log.Info("[REPO] Polling DNS seeds")
			repo.bootstrap()

		case addr := <-repo.addrDiscovered:
			n, ok := repo.nodeIndex[addr.String()]
			if ok {
				n.numSeen++
				continue
			}

			// Filter IPv6 for now
			ip := addr.IP.To4()
			if ip == nil {
				continue
			}

			// Broadcast
			rfc1700_1 := net.ParseIP("0.0.0.0")
			rfc1700_2 := net.ParseIP("0.255.255.255")
			if bytes.Compare(ip, rfc1700_1) >= 0 &&
				bytes.Compare(ip, rfc1700_2) <= 0 {
				continue
			}

			// LAN class A
			rfc1918_classa_1 := net.ParseIP("10.0.0.0")
			rfc1918_classa_2 := net.ParseIP("10.255.255.255")
			if bytes.Compare(ip, rfc1918_classa_1) >= 0 &&
				bytes.Compare(ip, rfc1918_classa_2) <= 0 {
				continue
			}

			// Carrier-grade NAT
			rfc6598_1 := net.ParseIP("100.64.0.0")
			rfc6598_2 := net.ParseIP("100.127.255.255")
			if bytes.Compare(ip, rfc6598_1) >= 0 &&
				bytes.Compare(ip, rfc6598_2) <= 0 {
				continue
			}

			// Loopback
			rfc990_1 := net.ParseIP("127.0.0.0")
			rfc990_2 := net.ParseIP("127.255.255.255")
			if bytes.Compare(ip, rfc990_1) >= 0 &&
				bytes.Compare(ip, rfc990_2) <= 0 {
				continue
			}

			// Link-local
			rfc3927_1 := net.ParseIP("169.254.0.0")
			rfc3927_2 := net.ParseIP("169.254.255.255")
			if bytes.Compare(ip, rfc3927_1) >= 0 &&
				bytes.Compare(ip, rfc3927_2) <= 0 {
				continue
			}

			// LAN class B
			rfc1918_classb_1 := net.ParseIP("172.16.0.0")
			rfc1918_classb_2 := net.ParseIP("172.32.255.255")
			if bytes.Compare(ip, rfc1918_classb_1) >= 0 &&
				bytes.Compare(ip, rfc1918_classb_2) <= 0 {
				continue
			}

			// Special Purpose Address Registry
			rfc5736_1 := net.ParseIP("192.0.0.0")
			rfc5736_2 := net.ParseIP("192.0.0.255")
			if bytes.Compare(ip, rfc5736_1) >= 0 &&
				bytes.Compare(ip, rfc5736_2) <= 0 {
				continue
			}

			// TEST-NET
			rfc5737_testnet_1 := net.ParseIP("192.0.2.0")
			rfc5737_testnet_2 := net.ParseIP("192.0.2.255")
			if bytes.Compare(ip, rfc5737_testnet_1) >= 0 &&
				bytes.Compare(ip, rfc5737_testnet_2) <= 0 {
				continue
			}

			// 6to4 anycast relays
			rfc3068_1 := net.ParseIP("192.88.99.0")
			rfc3068_2 := net.ParseIP("192.88.99.255")
			if bytes.Compare(ip, rfc3068_1) >= 0 &&
				bytes.Compare(ip, rfc3068_2) <= 0 {
				continue
			}

			// LAN class C
			rfc1918_classc_1 := net.ParseIP("192.168.0.0")
			rfc1918_classc_2 := net.ParseIP("192.168.255.255")
			if bytes.Compare(ip, rfc1918_classc_1) >= 0 &&
				bytes.Compare(ip, rfc1918_classc_2) <= 0 {
				continue
			}

			// Inter-network communications testing
			rfc2544_1 := net.ParseIP("198.18.0.0")
			rfc2544_2 := net.ParseIP("198.19.255.255")
			if bytes.Compare(ip, rfc2544_1) >= 0 &&
				bytes.Compare(ip, rfc2544_2) <= 0 {
				continue
			}

			// TEST-NET-2
			rfc5737_testnet2_1 := net.ParseIP("198.51.100.0")
			rfc5737_testnet2_2 := net.ParseIP("198.51.100.255")
			if bytes.Compare(ip, rfc5737_testnet2_1) >= 0 &&
				bytes.Compare(ip, rfc5737_testnet2_2) <= 0 {
				continue
			}

			// TEST-NET-3
			rfc5737_testnet3_1 := net.ParseIP("203.0.113.0")
			rfc5737_testnet3_2 := net.ParseIP("203.0.113.255")
			if bytes.Compare(ip, rfc5737_testnet3_1) >= 0 &&
				bytes.Compare(ip, rfc5737_testnet3_2) <= 0 {
				continue
			}

			// MULTICAST
			rfc5771_1 := net.ParseIP("224.0.0.0")
			rfc5771_2 := net.ParseIP("239.255.255.255")
			if bytes.Compare(ip, rfc5771_1) >= 0 &&
				bytes.Compare(ip, rfc5771_2) <= 0 {
				continue
			}

			// Reserved
			rfc6890_1 := net.ParseIP("240.0.0.0")
			rfc6890_2 := net.ParseIP("255.255.255.255")
			if bytes.Compare(ip, rfc6890_1) >= 0 &&
				bytes.Compare(ip, rfc6890_2) <= 0 {
				continue
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
