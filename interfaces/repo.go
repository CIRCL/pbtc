package interfaces

import (
	"net"

	"github.com/CIRCL/pbtc/domain"
)

type Repository struct {
	db map[net.TCPAddr]*Node
}

func (repo *Repository) AddNode(addr net.TCPAdr) {
	_, ok := repo.db[addr]
	if ok {
		return
	}

	node := &domain.Node{
		addr:      addr,
		connected: 0,
	}

	repo.db[addr] = node
}

func (repo *Repository) GetNodes(limit uint32) []*Node {
	var nodes []*Node

	for _, node := range repo.db {
		if atomic.LoadUint32(&node.active) != 0 {
			continue
		}

		nodes = append(nodes, node)

		if len(nodes) >= limit {
			break
		}
	}

	return nodes
}

func (repo *Repository) Connected(node *domain.Node) {
	node, ok := repo.db[node.addr]
	if !ok {
		return
	}

	atomic.StoreUint32(&node.active, 1)
}

func (repo *Repository) Disconnected(node *domain.Node) {
	_, ok := repo.db[node.addr]
	if !ok {
		return
	}

	atomic.StoreUint32(&node.active, 0)
}
