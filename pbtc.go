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

	// set logging levels
	logging.SetLevel(logging.INFO, "main")
	logging.SetLevel(logging.INFO, "repo")
	logging.SetLevel(logging.INFO, "rec")
	logging.SetLevel(logging.INFO, "mgr")
	logging.SetLevel(logging.INFO, "peer")

	// initialize console backend
	console, err := logger.NewConsoleBackend(
		logger.SetConsoleLevel(logging.INFO),
	)
	if err != nil {
		os.Exit(1)
	}

	// initialize file backend
	file, err := logger.NewFileBackend(
		logger.SetFileLevel(logging.DEBUG),
	)

	// register backends with logging library
	logging.SetBackend(console.Raw(), file.Raw())

	// start logging
	log := logging.MustGetLogger("main")
	log.Info("[PBTC] Starting modules")

	// repository
	repo, err := repository.New(
		repository.SetLogger(logging.MustGetLogger("repo")),
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
		recorder.SetLogger(logging.MustGetLogger("rec")),
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
		manager.SetLogger(logging.MustGetLogger("mgr")),
		manager.SetPeerLogger(logging.MustGetLogger("peer")),
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

	file.Cleanup()
	console.Cleanup()
	log.Info("[PBTC] All modules shutdown complete")

	os.Exit(0)
}
