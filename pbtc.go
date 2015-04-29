package main

import (
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/manager"
	"github.com/CIRCL/pbtc/recorder"
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
		logger.SetConsoleLevel(logging.INFO),
		logger.EnableFile(),
		logger.SetFileLevel(logging.DEBUG),
	)
	if err != nil {
		os.Exit(1)
	}

	// repository
	repo, err := repository.New(
		repository.SetLogger(log),
		repository.SetSeeds("seed.bitcoin.sipa.be"),
		repository.SetDefaultPort(8333),
	)
	if err != nil {
		log.Critical("Unable to create repository (%v)", err)
		os.Exit(1)
	}

	// recorder
	rec, err := recorder.New(
		recorder.SetLogger(log),
		recorder.SetTypes(wire.CmdTx, wire.CmdVersion, wire.CmdInv,
			wire.CmdAddr),
		recorder.SetFileSize(0),
		recorder.SetFileAge(time.Minute*5),
		recorder.EnableReset(),
	)
	if err != nil {
		log.Critical("Unable to initialize recorder (%v)", err)
		os.Exit(1)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLogger(log),
		manager.SetRepository(repo),
		manager.SetRecorder(rec),
		manager.SetNetwork(wire.MainNet),
		manager.SetVersion(wire.RejectVersion),
		manager.SetConnectionRate(time.Second/25),
		manager.SetInformationRate(time.Second*2),
	)
	if err != nil {
		log.Critical("Unable to create manager (%v)", err)
		os.Exit(1)
	}

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		log.Notice("Signal caught (%v)", sig.String())

		switch sig {
		case syscall.SIGINT:
			break SigLoop
		}
	}

	mgr.Stop()
	repo.Stop()

	os.Exit(0)
}
