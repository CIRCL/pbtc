package all

import (
	"encoding/gob"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

type repository struct {
	nodeIndex map[string]*node
}

type node struct {
	addr        *net.TCPAddr
	src         *net.TCPAddr
	attempts    uint32
	lastAttempt time.Time
	lastSuccess time.Time
}

func NewRepository() *repository {
	repo := &repository{
		nodeIndex: make(map[string]*node),
	}

	return repo
}

func NewNode(addr *net.TCPAddr, src *net.TCPAddr) *node {
	n := &node{
		addr: addr,
		src:  src,
	}

	return n
}

// Add will add new addresses to the repository. If the address is
// already known, it will update the relevant information.
func (repo *repository) Add(addr *net.TCPAddr, src *net.TCPAddr) {
	n := repo.find(addr)

	if n != nil {
		n.src = src
		return
	}

	if len(repo.nodeIndex) >= maxNodesTotal {
		return
	}

	n = NewNode(addr, src)
	repo.nodeIndex[addr.String()] = n
}

// Attempt will update the last connection attempt on the given address
// and increase the attempt counter accordingly.
func (repo *repository) Attempt(addr *net.TCPAddr) {
	n := repo.find(addr)

	if n == nil {
		return
	}

	n.attempts++
	n.lastAttempt = time.Now()
}

// Good will tag the connection to a given address as working correctly.
// It will also reset the attempt counter.
func (repo *repository) Good(addr *net.TCPAddr) {
	n := repo.find(addr)

	if n == nil {
		return
	}

	now := time.Now()
	n.lastAttempt = now
	n.lastSuccess = now
	n.attempts = 0
}

// Get will return one node that can currently be connected to.
func (repo *repository) Get() *net.TCPAddr {
	index := rand.Int() % len(repo.nodeIndex)
	i := 0
	for _, node := range repo.nodeIndex {
		if i == index {
			return node.addr
		}

		i++
	}

	return nil
}

// find will perform a search for the net address and return a
// pointer to the node structure if found.
func (repo *repository) find(addr *net.TCPAddr) *node {
	n, ok := repo.nodeIndex[addr.String()]

	if !ok {
		return nil
	}

	return n
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
		log.Println("[REPO] Could not restore node index from disk!")
	}
}
