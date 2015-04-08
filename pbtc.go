package main

import (
	"log"
	"math/rand"
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

	// create everything
	repo := all.NewRepository()
	mgr := all.NewManager()

	// start everything
	repo.Start()
	mgr.Start(wire.TestNet3, wire.RejectVersion)

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
	repo.Stop()

	log.Println("PBTC STOPPED")

	os.Exit(0)
}
