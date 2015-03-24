package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/CIRCL/pbtc/all"
)

func main() {
	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// find all our interfaces & their ips to listen
	ips := all.FindIPs()

	// create list of dns seeds to bootstrap discovery
	seeds := []string{
		//"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	// initialize the modules
	dManager := all.NewDataManager()
	cHandler := all.NewConnectionHandler()
	nRepo := all.NewNodeRepository()
	aHandler := all.NewAcceptHandler()
	dAgent := all.NewDiscoveryAgent()

	// set up the data manager to retrieve and log data
	dManager.Start()

	// set up the connection handler to initiate outgoing connections
	peerToManager := dManager.GetPeerIn()
	cHandler.Start(peerToManager)

	// set up node repository to manage known nodes
	addrToConnector := cHandler.GetAddrIn()
	nRepo.Start(addrToConnector)

	// set up accept handler to listen & accept incoming connections
	connToConnector := cHandler.GetConnIn()
	aHandler.Start(connToConnector)

	// set up discovery agent to bootstrap discovery
	addrToRepository := nRepo.GetAddrIn()
	dAgent.Start(addrToRepository)

	// feed listening IPs to the accept handler to start listeners
	ipToAcceptor := aHandler.GetIpIn()
	for _, ip := range ips {
		ipToAcceptor <- ip
	}

	// feed dns seeds to discovery agent to start bootstrapping
	seedToDiscovery := dAgent.GetSeedIn()
	for _, seed := range seeds {
		seedToDiscovery <- seed
	}

	// wait for input
	_, _ = fmt.Scanln()

	// stop all modules
	dAgent.Stop()
	aHandler.Stop()
	nRepo.Stop()
	cHandler.Stop()
	dManager.Stop()

	return
}
