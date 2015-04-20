package repository

import (
	"encoding/gob"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/CIRCL/pbtc/logger"
)

// Repository is the module responsible for managing all known node addresses. It creates
// a node for every new address and keeps track of all necessary information require to
// evaluate the node quality / reliability. It also stores this information in a file
// and restores it on start.
type Repository struct {
	nodeIndex map[string]*node

	seeds      []string
	backupPath string

	log logger.Logger
}

// NewRepository creates a new repository with all necessary variables initialized.
func New(options ...func(repo *Repository)) (*Repository, error) {
	repo := &Repository{
		nodeIndex: make(map[string]*node),
	}

	for _, option := range options {
		option(repo)
	}

	return repo, nil
}

func SetLogger(log logger.Logger) func(*Repository) {
	return func(mem *Repository) {
		mem.log = log
	}
}

func SetSeeds(seeds []string) func(*Repository) {
	return func(mem *Repository) {
		mem.seeds = seeds
	}
}

func SetBackupPath(path string) func(*Repository) {
	return func(mem *Repository) {
		mem.backupPath = path
	}
}

// bootstrap will use a number of dns seeds to discover nodes.
func (repo *Repository) Bootstrap() {
	// iterate over the seeds and try to get the ips
	for _, seed := range repo.seeds {
		// check if we can look up the ip addresses
		ips, err := net.LookupIP(seed)
		if err != nil {
			continue
		}

		// range over the ips and add them to the repository
		for _, ip := range ips {
			// try creating a TCP address from the given IP and default port
			addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip.String(), strconv.Itoa(18333)))
			if err != nil {
				continue
			}

			// check if we already know this address, if so we skip
			_, ok := repo.nodeIndex[addr.String()]
			if ok {
				continue
			}

			// now we can use update to add the address to our repository
			repo.Discovered(addr)
		}
	}
}

// save will try to save all current nodes to a file on disk.
func (repo *Repository) Save() error {
	// create the file, truncating if it already exists
	file, err := os.Create("nodes.dat")
	if err != nil {
		return err
	}
	defer file.Close()

	// encode the entire index using gob outputting into file
	enc := gob.NewEncoder(file)
	err = enc.Encode(repo.nodeIndex)
	if err != nil {
		return err
	}

	return nil
}

// restore will try to load the previously saved node file.
func (repo *Repository) Load() error {
	// open the nodes file in read-only mode
	file, err := os.Open("nodes.dat")
	if err != nil {
		return err
	}
	defer file.Close()

	// decode the entire index using gob reading from the file
	dec := gob.NewDecoder(file)
	err = dec.Decode(&repo.nodeIndex)
	if err != nil {
		return err
	}

	return nil
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
	return nil
}
