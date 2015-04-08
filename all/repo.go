package all

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type repository struct {
	nodeIndex map[string]*node
	sigSave   chan struct{}
	sigNode   chan struct{}
	addrQ     chan *net.TCPAddr
	nodeQ     chan *node
	wg        *sync.WaitGroup
	state     uint32
}

type node struct {
	addr        *net.TCPAddr
	src         *net.TCPAddr
	attempts    uint32
	lastAttempt time.Time
	lastSuccess time.Time
	lastConnect time.Time
}

func NewRepository() *repository {
	repo := &repository{
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

func newNode(addr *net.TCPAddr, src *net.TCPAddr) *node {
	n := &node{
		addr: addr,
		src:  src,
	}

	return n
}

func (node *node) GobEncode() ([]byte, error) {
	buffer := &bytes.Buffer{}
	enc := gob.NewEncoder(buffer)

	err := enc.Encode(node.addr)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.src)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.attempts)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastAttempt)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastSuccess)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastConnect)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (node *node) GobDecode(buf []byte) error {
	buffer := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&node.addr)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.src)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.attempts)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastAttempt)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastSuccess)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastConnect)
	if err != nil {
		return err
	}

	return nil
}

func (repo *repository) Start() {
	if !atomic.CompareAndSwapUint32(&repo.state, stateIdle, stateBusy) {
		return
	}

	repo.restore()

	repo.wg.Add(2)
	go repo.handleNodes()
	go repo.handleSave()

	repo.bootstrap()

	atomic.StoreUint32(&repo.state, stateRunning)

}

func (repo *repository) Stop() {
	if !atomic.CompareAndSwapUint32(&repo.state, stateRunning, stateBusy) {
		return
	}

	close(repo.sigSave)
	repo.save()

	atomic.StoreUint32(&repo.state, stateIdle)
}

// Add will add new addresses to the repository. If the address is
// already known, it will update the relevant information.
func (repo *repository) Update(addr *net.TCPAddr, src *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if ok {
		n.src = src
		return
	}

	n = newNode(addr, src)
	repo.nodeQ <- n
}

// Attempt will update the last connection attempt on the given address
// and increase the attempt counter accordingly.
func (repo *repository) Attempt(addr *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	n.attempts++
	n.lastAttempt = time.Now()
}

// Connected will tag the connection as currently connected.
func (repo *repository) Connected(addr *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	n.lastConnect = time.Now()
}

// Good will tag the connection to a given address as working correctly.
// It will also reset the attempt counter.
func (repo *repository) Good(addr *net.TCPAddr) {
	n, ok := repo.nodeIndex[addr.String()]
	if !ok {
		return
	}

	n.attempts = 0
	n.lastSuccess = time.Now()
}

// Get will return one node that can currently be connected to.
func (repo *repository) Get() (*net.TCPAddr, error) {
	if len(repo.nodeIndex) == 0 {
		return nil, errors.New("No nodes in repository")
	}

	index := rand.Int() % len(repo.nodeIndex)
	i := 0
	for _, node := range repo.nodeIndex {
		if i == index {
			return node.addr, nil
		}

		i++
	}

	return nil, errors.New("No qualified node found")
}

// save will try to save all current nodes to a file on disk
func (repo *repository) save() {
	file, err := os.Create("nodes.dat")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	err = enc.Encode(repo.nodeIndex)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Node index saved to file", len(repo.nodeIndex))
}

// restore will try to load the previously saved node file
func (repo *repository) restore() {
	file, err := os.Open("nodes.dat")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	err = dec.Decode(&repo.nodeIndex)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Node index restored from file", len(repo.nodeIndex))
}

func (repo *repository) bootstrap() {

	seeds := []string{
		"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	log.Println("Bootstrapping from DNS seeds", len(seeds))

	for _, seed := range seeds {
		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Println("Could not look up IPs", seed)
			continue
		}

		log.Println("Looked up IPs", seed, len(ips))

		for _, ip := range ips {
			port := GetDefaultPort()

			addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
			if err != nil {
				continue
			}

			_, ok := repo.nodeIndex[addr.String()]
			if ok {
				continue
			}

			repo.Update(addr, repo.local(addr))
		}
	}

	log.Println("Finished bootstrapping from DNS seeds")
}

func (repo *repository) local(addr *net.TCPAddr) *net.TCPAddr {
	local := &net.TCPAddr{}

	if addr.IP.To4() != nil {
		local.IP = net.IPv4zero
	} else {
		local.IP = net.IPv6zero
	}

	return local
}

func (repo *repository) handleSave() {
	defer repo.wg.Done()

	saveTicker := time.NewTicker(time.Second * 15)

SaveLoop:
	for {
		select {
		case _, ok := <-repo.sigSave:
			if !ok {
				break SaveLoop
			}

		case <-saveTicker.C:
			repo.save()
		}
	}
}

func (repo *repository) handleNodes() {
	defer repo.wg.Done()

NodeLoop:
	for {
		select {
		case _, ok := <-repo.sigNode:
			if !ok {
				break NodeLoop
			}

		case node, ok := <-repo.nodeQ:
			if !ok {
				repo.Stop()
			}

			_, ok = repo.nodeIndex[node.addr.String()]
			if ok {
				return
			}

			if len(repo.nodeIndex) >= maxNodeCount {
				return
			}

			repo.nodeIndex[node.addr.String()] = node
		}
	}
}
