package main

import (
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/wire"

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

	// create everything
	mgr := all.NewManager()
	svr := all.NewServer()
	dsc := all.NewDiscovery()

	// start everything
	mgr.Start(wire.TestNet3, wire.RejectVersion)
	svr.Start(mgr.GetConnIn())
	dsc.Start(mgr.GetAddrIn())

	// feed listen ips into server
	for _, ip := range ips {
		svr.GetAddrIn() <- net.JoinHostPort(ip, "18333")
	}

	// feed dns seeds into discovery
	for _, seed := range seeds {
		dsc.GetSeedIn() <- seed
	}

	// running
	_, _ = fmt.Scanln()

	// stop everything
	dsc.Stop()
	svr.Stop()
	mgr.Stop()

	return
}
