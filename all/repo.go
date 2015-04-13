package all

import (
	"encoding/gob"
	"errors"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/op/go-logging"
)

// Repository is the module responsible for managing all known node addresses. It creates
// a node for every new address and keeps track of all necessary information require to
// evaluate the node quality / reliability. It also stores this information in a file
// and restores it on start.
type Repository struct {
	nodeIndex map[string]*node
	sigSave   chan struct{}
	sigNode   chan struct{}
	addrQ     chan *net.TCPAddr
	nodeQ     chan *node
	wg        *sync.WaitGroup
	state     uint32
}

// NewRepository creates a new repository with all necessary variables initialized.
func NewRepository() *Repository {
	repo := &Repository{
		nodeIndex: make(map[string]*node),
		sigSave:   make(chan struct{}, 1),
		sigNode:   make(chan struct{}, 1),
		addrQ:     make(chan *net.TCPAddr, bufferRepoAddr),
		nodeQ:     make(chan *node, bufferRepoNode),
		wg:        &sync.WaitGroup{},
		state:     stateIdle,
	}

	return repo
}

// Start will restore our previous repository state from disk.
// It will also launch two handlers to handle new added nodes and to
// regularly save our nodes to disk. Finally, it will bootstrap the
// given DNS seeds in case we could not find nodes in our file.
func (repo *Repository) Start() {
	// we can only start the repository if we are in idle state
	if !atomic.CompareAndSwapUint32(&repo.state, stateIdle, stateBusy) {
		return
	}

	log := logging.MustGetLogger("pbtc")
	log.Info("Repository starting up...")

	// restore nodes from the disk
	err := repo.restore()
	if err != nil {
		log.Warning("Could not restore node index (%v)", err)
	}

	// add two handlers to waitgroup and launch them as goroutines
	repo.wg.Add(2)
	go repo.handleNodes()
	go repo.handleSave()

	// bootstrap ips from the known dns seeds
	repo.bootstrap()

	// at this point, we are up and running, so change the state
	atomic.StoreUint32(&repo.state, stateRunning)
}

// Stop will save all known nodes to disk after shutting down our handlers.
func (repo *Repository) Stop() {
	// we can only stop the repository if we are running
	if !atomic.CompareAndSwapUint32(&repo.state, stateRunning, stateBusy) {
		return
	}

	log := logging.MustGetLogger("pbtc")

	// signal our handlers to quit
	close(repo.sigSave)
	close(repo.sigNode)

	// save the nodes to disk one last time
	err := repo.save()
	if err != nil {
		log.Warning("Could not save node index (%v)", err)
	}

	// wait for handlers to return
	log.Debug("Waiting for handlers to quit")
	repo.wg.Wait()

	// we are not no longer running, so set new state
	log.Info("Repository shutdown complete")
	atomic.StoreUint32(&repo.state, stateIdle)
}

// Update will update the information of a given address in our repository.
// At this point, this is only the address that has last seen the node.
// If the node doesn't exist yet, we create one.
func (repo *Repository) Update(addr *net.TCPAddr, src *net.TCPAddr) {
	log := logging.MustGetLogger("pbtc")

	// check if a node with the given address already exists
	// if so, simply update the source address
	n, ok := repo.nodeIndex[addr.String()]
	if ok {
		log.Debug("%v: updated source address (%v -> %v)", n, n.src, src)
		n.src = src
		return
	}

	// if we don't know this address yet, create node and add to repo
	n = newNode(addr, src)
	log.Debug("%v: created new node", n)
	repo.nodeQ <- n
}

// Attempt will update the last connection attempt on the given address
// and increase the attempt counter accordingly.
func (repo *Repository) Attempt(addr *net.TCPAddr) {
	log := logging.MustGetLogger("pbtc")

	// if we don't know this address, ignore
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		log.Notice("%v: attempt on non-existing node", addr)
		return
	}

	// increase number of attempts and timestamp last attempt
	log.Debug("%v: tagged attempt", n)
	n.attempts++
	n.lastAttempt = time.Now()
}

// Connected will tag the connection as currently connected. This is used
// in the reference client to send timestamps with the addresses, but only
// maximum once every 20 minutes. We will not give out any such information,
// but it can still be useful to determine which addresses to try to connect to
// next.
func (repo *Repository) Connected(addr *net.TCPAddr) {
	log := logging.MustGetLogger("pbtc")

	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		log.Notice("%v: connected on non-existing node", addr)
		return
	}

	log.Debug("%v: tagged connected", n)
	n.lastConnect = time.Now()
}

// Good will tag the connection to a given address as working correctly.
// It is called after a successful handshake and will reset the attempt
// counter and timestamp last success. The reference client timestamps
// the other fields as well, but all we do with that is lose some extra
// information that we could use to choose our addresses.
func (repo *Repository) Good(addr *net.TCPAddr) {
	log := logging.MustGetLogger("pbtc")

	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		log.Notice("%v: good on non-existing node", addr)
		return
	}

	log.Debug("%v: tagged good", n)
	n.attempts = 0
	n.lastSuccess = time.Now()
}

// Get will return one node that can currently be connected to. It should
// do so by taking all kinds of factors into account, like how many nodes
// know this address, how many times we already tried/succeeded, how long
// ago we last saw/connected to the node, what the reputation of nodes is
// that we receive the address from.
func (repo *Repository) Get() (*net.TCPAddr, error) {
	// if we know no nodes, we return an error and nil value
	if len(repo.nodeIndex) == 0 {
		return nil, errors.New("no nodes in repository")
	}

	// for now, this simply picks a random node from our index
	for tries := 0; tries < 128; tries++ {
		node := repo.getRandomNode()
		if node == nil {
			continue
		}

		if node.attempts > 5 {
			continue
		}

		if node.lastAttempt.Add(time.Minute).After(time.Now()) {
			continue
		}

		return node.addr, nil
	}

	// we should never get here at this point
	return nil, errors.New("no qualified node found")
}

// getRandomNode selects a random node from the repository.
func (repo *Repository) getRandomNode() *node {
	index := rand.Int() % len(repo.nodeIndex)
	i := 0
	for _, node := range repo.nodeIndex {
		if i == index {
			return node
		}

		i++
	}

	return nil
}

// save will try to save all current nodes to a file on disk.
func (repo *Repository) save() error {
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
func (repo *Repository) restore() error {
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

// bootstrap will use a number of dns seeds to discover nodes.
func (repo *Repository) bootstrap() {
	log := logging.MustGetLogger("pbtc")

	// at this point, we simply define the seeds here
	seeds := []string{
		"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	log.Info("Bootstrapping from %v DNS seeds", len(seeds))

	// iterate over the seeds and try to get the ips
	for _, seed := range seeds {
		// check if we can look up the ip addresses
		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Warning("%v: could not look up IPs", seed)
			continue
		}

		log.Info("%v: %v IP(s) found", seed, len(ips))

		// range over the ips and add them to the repository
		for _, ip := range ips {
			// try creating a TCP address from the given IP and default port
			port := GetDefaultPort()
			addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
			if err != nil {
				log.Error("%v: could not resolve TCP address", ip)
				continue
			}

			// check if we already know this address, if so we skip
			_, ok := repo.nodeIndex[addr.String()]
			if ok {
				log.Debug("%v: node already known", addr)
				continue
			}

			// now we can use update to add the address to our repository
			log.Debug("%v: adding to repository", addr)
			repo.Update(addr, repo.local(addr))
		}
	}
}

// local will return the best local IP address to route to the given remote address.
func (repo *Repository) local(addr *net.TCPAddr) *net.TCPAddr {
	local := &net.TCPAddr{}

	// Right now, we simply return the zero address for either IPv4 or IPv6.
	if addr.IP.To4() != nil {
		local.IP = net.IPv4zero
	} else {
		local.IP = net.IPv6zero
	}

	return local
}

// handleSave is the handler to regularly save our node index to disk.
func (repo *Repository) handleSave() {
	log := logging.MustGetLogger("pbtc")
	log.Debug("Save handler started")

	// let the waitgroup know when we are done
	defer repo.wg.Done()

	// initialize the ticker to save nodes
	saveTicker := time.NewTicker(nodeSaveInterval)

SaveLoop:
	for {
		select {
		// signal to quit, so break outer loop
		case _, ok := <-repo.sigSave:
			if !ok {
				break SaveLoop
			}

		// each time this ticks, we should save our node index to disk
		case <-saveTicker.C:
			repo.save()
		}
	}

	log.Debug("Save handler stopped")
}

// handleNodes will take new added nodes and put them into the index.
func (repo *Repository) handleNodes() {
	log := logging.MustGetLogger("pbtc")
	log.Debug("Node handler started")

	// let the waitgroup know when we are done
	defer repo.wg.Done()

NodeLoop:
	for {
		select {
		// if we get the stop signal, break outer loop
		case _, ok := <-repo.sigNode:
			if !ok {
				break NodeLoop
			}

		case node := <-repo.nodeQ:
			// if we already know this address, skip
			_, ok := repo.nodeIndex[node.addr.String()]
			if ok {
				return
			}

			// if we have reached our limit of nodes, skip
			if len(repo.nodeIndex) >= maxNodeCount {
				return
			}

			// add the node to the index
			repo.nodeIndex[node.addr.String()] = node
		}
	}

	log.Debug("Node handler stopped")
}
