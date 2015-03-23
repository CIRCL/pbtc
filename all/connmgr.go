package all

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

type ConnMgr struct {
	listIn    chan string
	nodeIn    chan *Node
	ticker    *time.Ticker
	listeners []net.Listener
	nodes     []*Node
	Network   wire.BitcoinNet
	Version   uint32
	nonce     uint64
}

var netcfg = &chaincfg.TestNet3Params

func NewConnMgr(network wire.BitcoinNet, version uint32) (*ConnMgr, error) {
	nonce, err := wire.RandomUint64()
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not initialize server nonce")
	}

	var listeners []net.Listener
	var nodes []*Node

	connMgr := &ConnMgr{
		Network:   network,
		Version:   version,
		ticker:    time.NewTicker(pollInterval * time.Second),
		listeners: listeners,
		nodes:     nodes,
		nonce:     nonce,
	}

	return connMgr, nil
}

func (connMgr *ConnMgr) Start(listIn chan string, nodeIn chan *Node) {
	connMgr.listIn = listIn
	connMgr.nodeIn = nodeIn

	go connMgr.handleListeners()
	go connMgr.handleNodes()
	go connMgr.handlePoll()
}

func (connMgr *ConnMgr) Stop() {
	connMgr.ticker.Stop()

	for _, listener := range connMgr.listeners {
		listener.Close()
	}

	for _, node := range connMgr.nodes {
		node.Disconnect()
	}

	close(connMgr.listIn)
	close(connMgr.nodeIn)
}

func (connMgr *ConnMgr) AddListener(list string) {
	connMgr.listIn <- list
}

func (connMgr *ConnMgr) AddNode(node *Node) {
	connMgr.nodeIn <- node
}

func (connMgr *ConnMgr) handleListeners() {
	for list := range connMgr.listIn {
		listener, err := net.Listen("tcp", list+":"+netcfg.DefaultPort)
		if err != nil {
			log.Println(err)
			continue
		}

		connMgr.listeners = append(connMgr.listeners, listener)
		log.Println("Now listening on", len(connMgr.listeners), "IP(s)")
	}
}

func (connMgr *ConnMgr) handleNodes() {
	for node := range connMgr.nodeIn {
		connMgr.nodes = append(connMgr.nodes, node)
		log.Println("Now talking to", len(connMgr.nodes), "nodes")
	}

}

func (connMgr *ConnMgr) handlePoll() {
	for _ = range connMgr.ticker.C {
		log.Println("Polling for new nodes")

		for _, node := range connMgr.nodes {
			if node.IsConnected() {
				node.Poll()
			}
		}
	}
}
