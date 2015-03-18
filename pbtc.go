package main

import (
	"log"
	"net"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/domain"
	"github.com/CIRCL/pbtc/usecases"
)

func main() {
	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// initialize communication channels between modules
	connChan := make(chan net.Conn, 128)
	peerChan := make(chan *usecases.Peer, 128)
	nodeChan := make(chan string, 128)
	seedChan := make(chan string, 128)
	listChan := make(chan string, 128)

	// the server will handle all peers that are set up
	serv, err := usecases.NewServer(wire.TestNet3, wire.ProtocolVersion)
	if err != nil {
		log.Println(err)
	}

	// the discoverer will poll dns seeds for node ips
	disc, err := domain.NewDiscoverer()
	if err != nil {
		log.Println(err)
	}

	// the initializer will take care of connection handshakes
	init, err := domain.NewInitializer(serv.Version, serv.Network)
	if err != nil {
		log.Println(err)
	}

	// start all modules
	serv.Start(listChan, peerChan)
	init.Start(nodeChan, connChan, peerChan)
	disc.Start(seedChan, nodeChan)

	// give the server the ips to listen on
	ips := domain.FindIPs()

	for _, ip := range ips {
		serv.AddListener(ip)
	}

	// give the discoverer the dns seeds that we want to poll
	seeds := []string{
		//"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	for _, seed := range seeds {
		disc.AddSeed(seed)
	}

	// wait for a bit
	timer := time.NewTimer(120 * time.Second)
	<-timer.C

	// stop all module
	disc.Stop()
	init.Stop()
	serv.Stop()

	return
}
