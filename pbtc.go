package main

import (
	"log"
	"net"
	"runtime"
	"time"

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

	// the discoverer will poll dns seeds for node ips
	disc, err := domain.NewDiscoverer(seedChan, nodeChan)
	if err != nil {
		log.Println(err)
	}

	// the initializer will take care of connection handshakes
	init, err := domain.NewInitializer(nodeChan, connChan, peerChan)
	if err != nil {
		log.Println(err)
	}

	// the server will handle all peers that are set up
	serv, err := usecases.NewServer(listChan, peerChan)
	if err != nil {
		log.Println(err)
	}

	// start all modules
	serv.Start()
	disc.Start()
	init.Start()

	// give the server the ips to listen on
	ips := domain.FindIPs()

	for _, ip := range ips {
		listChan <- ip
	}

	// give the discoverer the dns seeds that we want to poll
	seeds := []string{
		//"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	for _, seed := range seeds {
		seedChan <- seed
	}

	// wait for a bit
	timer := time.NewTimer(120 * time.Second)
	<-timer.C

	// close all channels to end modules
	close(seedChan)

	return
}
