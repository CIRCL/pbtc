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
	dManager.Start(
		nRepo.GetAddrIn(),
	)

	// set up the connection handler to initiate outgoing connections
	cHandler.Start(
		dManager.GetPeerIn(),
	)

	// set up node repository to manage known nodes
	nRepo.Start(
		cHandler.GetAddrIn(),
	)

	// set up accept handler to listen & accept incoming connections
	aHandler.Start(
		cHandler.GetConnIn(),
	)

	// set up discovery agent to bootstrap discovery
	dAgent.Start(
		nRepo.GetAddrIn(),
	)

	// feed listening IPs to the accept handler to start listeners
	for _, ip := range ips {
		aHandler.GetIpIn() <- ip
	}

	// feed dns seeds to discovery agent to start bootstrapping
	for _, seed := range seeds {
		dAgent.GetSeedIn() <- seed
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
