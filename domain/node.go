package domain

import (
	"net"
)

type NodeRepository interface {
	AddNode(tcpAddr net.TCPAddr)
	FindByAddr(addr net.TCPAddr) *Node
}

type Node struct {
	addr net.TCPAddr
}
