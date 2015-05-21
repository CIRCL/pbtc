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

	"github.com/CIRCL/pbtc/compressor"
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

	log.Info("[PBTC] Starting modules")

	// repository
	repo, err := repository.New(
		repository.SetLogger(log),
		repository.SetSeeds("seed.bitcoin.sipa.be"),
		repository.SetDefaultPort(8333),
		repository.DisableRestore(),
	)
	if err != nil {
		log.Critical("Unable to create repository (%v)", err)
		os.Exit(2)
	}

	// recorder
	rec, err := recorder.New(
		recorder.SetLogger(log),
		recorder.SetSizeLimit(0),
		recorder.SetAgeLimit(time.Minute*5),
		recorder.SetCompressor(compressor.NewLZ4()),
	)
	if err != nil {
		log.Critical("Unable to initialize recorder (%v)", err)
		os.Exit(3)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLogger(log),
		manager.SetRepository(repo),
		manager.SetRecorder(rec),
		manager.SetNetwork(wire.MainNet),
		manager.SetVersion(wire.RejectVersion),
		manager.SetConnectionRate(time.Second/25),
		manager.SetInformationRate(time.Second*10),
		manager.SetPeerLimit(1000),
	)
	if err != nil {
		log.Critical("Unable to create manager (%v)", err)
		os.Exit(4)
	}

	log.Info("[PBTC] All modules initialization complete")

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		log.Notice("Signal caught (%v)", sig.String())

		switch sig {
		case syscall.SIGINT:
			break SigLoop
		}
	}

	// we will initialize shutdown in a non-blocking way
	c := make(chan struct{})
	go func() {
		mgr.Stop()
		repo.Stop()
		rec.Stop()
		c <- struct{}{}
	}()

	// if the shutdown completes, we simple quit normally
	// however, if we receive another signal during shutdown, we panic
	// this allows us to see the stacktrace in case shutdown blocks somewhere
	select {
	case <-sigc:
		panic("SHUTDOWN FAILED")

	case <-c:
		break
	}

	log.Info("[PBTC] All modules shutdown complete")

	os.Exit(0)
}
