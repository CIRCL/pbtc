package all

import (
	"errors"
	"log"
	"net"
)

type NodeMgr struct {
	connMgr  *ConnMgr
	nodeList map[string]*Node

	seedIn  chan string
	peerIn  chan *Node
	addrEx  chan string
	nodeOut chan *Node
}

func NewNodeMgr(connMgr *ConnMgr) (*NodeMgr, error) {
	if connMgr == nil {
		return nil, errors.New("Can't create NodeMgr with nil ConnMgr")
	}

	nodeList := make(map[string]*Node)

	nodeMgr := &NodeMgr{
		connMgr:  connMgr,
		nodeList: nodeList,
	}

	return nodeMgr, nil
}

func (nodeMgr *NodeMgr) Start(seedIn chan string, peerIn chan *Node, nodeOut chan *Node) {
	nodeMgr.seedIn = seedIn
	nodeMgr.peerIn = peerIn

	nodeMgr.addrEx = make(chan string, 128)

	nodeMgr.nodeOut = nodeOut

	go nodeMgr.handleSeeds()
	go nodeMgr.handleAddresses()
}

func (nodeMgr *NodeMgr) Stop() {
	close(nodeMgr.nodeOut)
	close(nodeMgr.addrEx)
}

func (nodeMgr *NodeMgr) AddSeed(seed string) {
	nodeMgr.seedIn <- seed
}

func (nodeMgr *NodeMgr) AddPeer(node *Node) {
	nodeMgr.peerIn <- node
}

func (nodeMgr *NodeMgr) handleSeeds() {
	for seed := range nodeMgr.seedIn {
		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Println(err)
			return
		}

		for _, ip := range ips {
			addr := net.JoinHostPort(ip.String(), netcfg.DefaultPort)
			log.Println("Discovered node at", addr)
			nodeMgr.addrEx <- addr
		}
	}
}

func (nodeMgr *NodeMgr) handleAddresses() {
	for addr := range nodeMgr.addrEx {
		_, ok := nodeMgr.nodeList[addr]
		if !ok {
			node, err := NewNode(addr, nodeMgr.addrEx)
			if err != nil {
				log.Println("Could not initialize new node", addr, err)
				continue
			}

			nodeMgr.nodeList[addr] = node
		}

		node := nodeMgr.nodeList[addr]
		if node.IsIdle() && len(nodeMgr.nodeList) <= 1024 {
			log.Println("Connecting to node at", addr)
			go node.Connect(nodeMgr.connMgr)
		}
	}
}
