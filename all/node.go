package all

import (
	"log"
	"math"
	"math/rand"
	"net"
	"time"
)

type node struct {
	addr    string
	backoff float64
}

func NewNode(addr string) *node {

	node := &node{
		addr:    addr,
		backoff: backoffInitial,
	}

	return node
}

func (node *node) Dial(addrOut chan<- string, connOut chan<- net.Conn) {

	conn, err := net.DialTimeout("tcp", node.addr, timeoutDial)
	if err != nil {
		log.Println("Connection failed:", node.addr, err)
		addrOut <- node.addr
		return
	}

	log.Println("Connection successful:", node.addr)
	node.backoff = backoffInitial
	connOut <- conn
}

func (node *node) Retry(nodeOut chan<- *node) {

	backoff := time.Duration(int32((backoffRandomizer*rand.Float64()+1.0)*node.backoff)) * time.Second
	node.backoff = math.Min(backoffMultiplier*node.backoff, backoffMaximum)

	log.Println("Retrying in:", backoff)
	timer := time.NewTimer(backoff)
	<-timer.C
	nodeOut <- node
}
