package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CIRCL/pbtc/all"

	"github.com/btcsuite/btcd/wire"
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

	// initialize our modules
	repo := all.NewRepository()
	mgr := all.NewManager()

	// start our modules
	repo.Start()
	mgr.Start(repo, wire.TestNet3, wire.RejectVersion)

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case os.Interrupt:
			log.Println("PBTC SHUTTING DOWN")
			break SigLoop

		case syscall.SIGTERM:

		case syscall.SIGHUP:

		case syscall.SIGINT:

		case syscall.SIGQUIT:
		}
	}

	// stop our modules
	mgr.Stop()
	repo.Stop()

	log.Println("PBTC CLOSING")

	os.Exit(0)
}
