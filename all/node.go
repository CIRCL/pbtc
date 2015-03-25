package all

import (
	"net"
)

const (
	stateIdle = iota
	statePending
	stateConnected
)

type node struct {
	Addr  string
	state uint32
	conn  net.Conn
}

func NewNode(addr string) *node {
	node := &node{
		Addr:  addr,
		state: stateIdle,
	}

	return node
}

func (node *node) UseConnection(conn net.Conn) {
	node.state = statePending
	node.conn = conn
}

func (node *node) GetState() uint32 {
	return node.state
}
