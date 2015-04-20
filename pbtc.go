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
		log.Critical("Unable to create repositor (%v)", err)
		os.Exit(1)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLogger(log),
		manager.SetRepository(repo),
	)
	if err != nil {
		log.Critical("Unable to create manager (%v)", err)
		os.Exit(1)
	}

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case syscall.SIGINT:
			break SigLoop

		default:
			log.Notice("Signal caught (%v)", sig.String())
		}
	}

	mgr.Cleanup()
	repo.Cleanup()

	os.Exit(0)
}
