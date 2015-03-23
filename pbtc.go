package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/all"
)

func main() {
	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// initialize communication channels between modules
	listChan := make(chan string, 128)
	seedChan := make(chan string, 128)
	peerChanToConn := make(chan *all.Node, 128)
	peerChanToNode := make(chan *all.Node, 128)

	// the server will handle all peers that are set up
	connMgr, err := all.NewConnMgr(wire.TestNet3, wire.ProtocolVersion)
	if err != nil {
		log.Println(err)
	}

	// the discoverer will poll dns seeds for node ips
	nodeMgr, err := all.NewNodeMgr(connMgr)
	if err != nil {
		log.Println(err)
	}

	// start all modules
	connMgr.Start(listChan, peerChanToConn)
	nodeMgr.Start(seedChan, peerChanToNode, peerChanToConn)

	// give the server the ips to listen on
	ips := all.FindIPs()

	for _, ip := range ips {
		connMgr.AddListener(ip)
	}

	// give the discoverer the dns seeds that we want to poll
	seeds := []string{
		//"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	for _, seed := range seeds {
		nodeMgr.AddSeed(seed)
	}

	// wait for input
	_, _ = fmt.Scanln()

	// stop all module
	nodeMgr.Stop()
	connMgr.Stop()

	return
}
