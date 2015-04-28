package repository

import (
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
	wg              *sync.WaitGroup
	backupSig       chan struct{}
	backupTicker    *time.Ticker
	bootstrapTicker *time.Ticker
	nodeIndex       map[string]*node

	seeds      []string
	backupPath string

	log adaptor.Logger

	done uint32
}

// New creates a new repository initialized with default values. A variable list
// of options can be provided to override default behaviour.
func New(options ...func(repo *Repository)) (*Repository, error) {
	repo := &Repository{
		wg:              &sync.WaitGroup{},
		nodeIndex:       make(map[string]*node),
		backupSig:       make(chan struct{}, 1),
		backupTicker:    time.NewTicker(90 * time.Second),
		bootstrapTicker: time.NewTicker(30 * time.Minute),

		seeds:      []string{"testnet-seed.bitcoin.petertodd.org"},
		backupPath: "nodes.dat",
	}

	for _, option := range options {
		option(repo)
	}

	//repo.restore()

	if len(repo.nodeIndex) == 0 {
		repo.bootstrap()
	}

	repo.start()

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

// Cleanup is used to clean up all resources associated with a repository
// instance. It will return once all go routines have ended and all resources
// are ready to be collected by the GC.
func (repo *Repository) Cleanup() {
	repo.shutdown()
}

// Update will update the information of a given address in our repository.
// At this point, this is only the address that has last seen the node.
// If the node doesn't exist yet, we create one.
func (repo *Repository) Discovered(addr *net.TCPAddr) {
	// check if a node with the given address already exists
	// if so, simply update the source address
	n, ok := repo.nodeIndex[addr.String()]
	if ok {
		return
	}

	// if we don't know this address yet, create node and add to repo
	n = newNode(addr)
	repo.nodeIndex[addr.String()] = n
}

// Attempt will update the last connection attempt on the given address
// and increase the attempt counter accordingly.
func (repo *Repository) Attempted(addr *net.TCPAddr) {
	// if we don't know this address, ignore
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	// increase number of attempts and timestamp last attempt
	n.numAttempts++
	n.lastAttempted = time.Now()
}

// Connected will tag the connection as currently connected. This is used
// in the reference client to send timestamps with the addresses, but only
// maximum once every 20 minutes. We will not give out any such information,
// but it can still be useful to determine which addresses to try to connect to
// next.
func (repo *Repository) Connected(addr *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	n.lastConnected = time.Now()
}

// Good will tag the connection to a given address as working correctly.
// It is called after a successful handshake and will reset the attempt
// counter and timestamp last success. The reference client timestamps
// the other fields as well, but all we do with that is lose some extra
// information that we could use to choose our addresses.
func (repo *Repository) Succeeded(addr *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	n.numAttempts = 0
	n.lastSucceeded = time.Now()
}

func (repo *Repository) Retrieve() *net.TCPAddr {
	for _, node := range repo.nodeIndex {
		if node.numAttempts >= 3 {
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

		return node.addr
	}

	return nil
}

func (repo *Repository) start() {
	repo.wg.Add(1)
	go repo.goBackups()
}

func (repo *Repository) shutdown() {
	if atomic.SwapUint32(&repo.done, 1) == 1 {
		return
	}

	repo.bootstrapTicker.Stop()
	repo.backupTicker.Stop()
	close(repo.backupSig)
	repo.save()
	repo.wg.Wait()
}

// bootstrap will use a number of dns seeds to discover nodes.
func (repo *Repository) bootstrap() {
	count, failed, known := 0, 0, 0

	repo.log.Info("bootstrapping from %v dns seeds", len(repo.seeds))

	// iterate over the seeds and try to get the ips
	for _, seed := range repo.seeds {
		// check if we can look up the ip addresses
		ips, err := net.LookupIP(seed)
		if err != nil {
			repo.log.Notice("could not lookup %v (%v)", seed, err)
			continue
		}

		repo.log.Info("%v: found %v ips", seed, len(ips))

		// range over the ips and add them to the repository
		for _, ip := range ips {
			addr := &net.TCPAddr{IP: ip, Port: 18333}

			_, ok := repo.nodeIndex[addr.String()]
			if ok {
				known++
				continue
			}

			// now we can use update to add the address to our repository
			count++
			repo.Discovered(addr)
		}
	}

	repo.log.Info("added %v nodes from bootstrap (%v failed, %v known)",
		count, failed, known)
}

// save will try to save all current nodes to a file on disk.
func (repo *Repository) save() {
	// create the file, truncating if it already exists
	file, err := os.Create(repo.backupPath)
	if err != nil {
		repo.log.Warning("could not save backup (%v)", err)
		return
	}
	defer file.Close()

	// encode the entire index using gob outputting into file
	enc := gob.NewEncoder(file)
	err = enc.Encode(repo.nodeIndex)
	if err != nil {
		repo.log.Warning("could not encode backup (%v)", err)
		return
	}

	repo.log.Info("saved %v nodes to backup", len(repo.nodeIndex))
}

// restore will try to load the previously saved node file.
func (repo *Repository) restore() {
	// open the nodes file in read-only mode
	file, err := os.Open(repo.backupPath)
	if err != nil {
		repo.log.Warning("could not restore backup (%v)", err)
		return
	}
	defer file.Close()

	// decode the entire index using gob reading from the file
	dec := gob.NewDecoder(file)
	err = dec.Decode(&repo.nodeIndex)
	if err != nil {
		repo.log.Warning("could not decode backup (%v)", err)
		return
	}

	repo.log.Info("restored %v nodes from backup", len(repo.nodeIndex))
}

func (repo *Repository) goBackups() {
	defer repo.wg.Done()

	for atomic.LoadUint32(&repo.done) != 1 {
		select {
		case _, ok := <-repo.backupSig:
			if !ok {
				repo.shutdown()
			}

		case <-repo.backupTicker.C:
			repo.save()

		case <-repo.bootstrapTicker.C:
			repo.bootstrap()
		}
	}
}
