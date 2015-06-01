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
	"github.com/CIRCL/pbtc/filter"
	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/manager"
	"github.com/CIRCL/pbtc/repository"
	"github.com/CIRCL/pbtc/writer"
)

func main() {
	// catch signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	signal.Notify(sigc, syscall.SIGHUP)

	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// initialize logging
	logr, err := logger.NewGologging(
		logger.EnableConsole(),
		logger.SetConsoleLevel(logging.INFO),
		logger.EnableFile(),
		logger.SetFileLevel(logging.DEBUG),
		logger.SetFilePath("pbtc.log"),
		logger.SetLevel("main", logging.INFO),
		logger.SetLevel("repo", logging.INFO),
		logger.SetLevel("rec", logging.INFO),
		logger.SetLevel("mgr", logging.INFO),
		logger.SetLevel("peer", logging.INFO),
	)
	if err != nil {
		os.Exit(1)
	}

	// start logging
	log := logr.GetLog("main")
	log.Info("[PBTC] Starting modules")

	// repository
	repo, err := repository.New(
		repository.SetLog(logr.GetLog("repo")),
		repository.SetSeeds("seed.bitcoin.sipa.be"),
		repository.SetDefaultPort(8333),
		repository.DisableRestore(),
	)
	if err != nil {
		log.Critical("Unable to create repository (%v)", err)
		os.Exit(2)
	}

	// writer to write everything to file
	wfile, err := writer.NewFile(
		writer.SetLogFile(logr.GetLog("out")),
		writer.SetSizeLimit(0),
		writer.SetAgeLimit(time.Minute*5),
		writer.SetCompressor(compressor.NewLZ4()),
		writer.SetFilePath("logs/"),
	)
	if err != nil {
		log.Critical("Unable to initialize file writer (%v)", err)
		os.Exit(3)
	}

	// writer to publish stuff on zeromq
	wzmq, err := writer.NewZMQ(
		writer.SetLogZMQ(logr.GetLog("out")),
		writer.SetAddressZMQ("127.0.0.1:12345"),
	)
	if err != nil {
		log.Critical("Unable to initialize zeromq writer (%v)", err)
		os.Exit(3)
	}

	// writer to publish stuff to redis
	/*wredis, err := writer.NewRedis(
		writer.SetLogRedis(logr.GetLog("out")),
		writer.SetAddressRedis("127.0.0.1:23456"),
		writer.SetPassword(""),
		writer.SetDatabase(0),
	)
	if err != nil {
		log.Critical("Unable to initialize redis writer (%v)", err)
		os.Exit(3)
	}*/

	// recorder that doesn't filter
	rec_all, err := filter.New(
		filter.SetLog(logr.GetLog("rec")),
		filter.AddWriter(wfile),
		filter.AddWriter(wzmq),
		//filter.AddWriter(wredis),
	)
	if err != nil {
		log.Critical("Unable to initialize full filter (%v)", err)
		os.Exit(4)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLog(logr.GetLog("mgr")),
		manager.SetPeerLog(logr.GetLog("peer")),
		manager.SetRepository(repo),
		manager.AddFilter(rec_all),
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

		case syscall.SIGHUP:
			// reload config
			continue
		}
	}

	// we will initialize shutdown in a non-blocking way
	c := make(chan struct{})
	go func() {
		c <- struct{}{}
	}()

	// if the shutdown completes, we simple quit normally
	// however, if we receive another signal during shutdown, we panic
	// this allows us to see the stacktrace in case shutdown blocks somewhere
	select {
	case <-sigc:
		panic("SHUTDOWN FAILED")

	case <-c:
		mgr.Close()
		break
	}

	log.Info("[PBTC] All modules shutdown complete")

	os.Exit(0)
}
