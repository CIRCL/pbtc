package domain

import (
	"net"
)

type NodeRepository interface {
	AddNode(addr net.TCPAddr)
	GetNodes(limit uint32) []*Node
	Connected(node *Node)
	Disconnected(node *Node)
}

type Node struct {
	addr   net.TCPAddr
	active uint32
}
