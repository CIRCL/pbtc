package main

import (
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/manager"
	"github.com/CIRCL/pbtc/repository"
)

func main() {
	// catch signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// logging
	log, err := logger.New(
		logger.EnableConsole(),
		logger.SetConsoleLevel(logging.DEBUG),
		logger.EnableFile(),
		logger.SetFileLevel(logging.INFO),
	)
	if err != nil {
		os.Exit(1)
	}

	// repository
	repo, err := repository.New(
		repository.SetLogger(log),
		repository.SetSeeds([]string{"testnet-seed.alexykot.me",
			"testnet-seed.bitcoin.petertodd.org",
			"testnet-seed.bluematt.me",
			"testnet-seed.bitcoin.schildbach.de"}),
	)
	if err != nil {
		log.Critical("%v", err)
		os.Exit(2)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLogger(log),
		manager.SetRepository(repo),
	)
	if err != nil {
		log.Critical("%v", err)
		os.Exit(3)
	}

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case os.Interrupt:
			break SigLoop

		case syscall.SIGTERM:

		case syscall.SIGHUP:

		case syscall.SIGINT:

		case syscall.SIGQUIT:
		}
	}

	mgr.Stop()
	os.Exit(0)
}
