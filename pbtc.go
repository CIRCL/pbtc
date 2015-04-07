package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/all"
)

func main() {
	log.Println("PBTC STARTING")

	// catch signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

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

	// check for signals
SigLoop:
	for sig := range sigc {
		switch sig {
		case os.Interrupt:
			log.Println("PBTC STOPPING")
			break SigLoop

		case syscall.SIGTERM:

		case syscall.SIGHUP:

		case syscall.SIGINT:

		case syscall.SIGQUIT:
		}
	}

	// stop everything
	dsc.Stop()
	svr.Stop()
	mgr.Stop()

	log.Println("PBTC STOPPED")

	os.Exit(0)
}
