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
	"github.com/CIRCL/pbtc/loglib"
	"github.com/CIRCL/pbtc/manager"
	"github.com/CIRCL/pbtc/processor"
	"github.com/CIRCL/pbtc/repository"
	"github.com/CIRCL/pbtc/server"
	"github.com/CIRCL/pbtc/supervisor"
	"github.com/CIRCL/pbtc/tracker"
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
	logr, err := loglib.NewGologging(
		loglib.EnableConsole(),
		loglib.SetConsoleLevel(logging.INFO),
		loglib.EnableFile(),
		loglib.SetFileLevel(logging.DEBUG),
		loglib.SetFilePath("pbtc.log"),
		loglib.SetLevel("main", logging.INFO),
		loglib.SetLevel("repo", logging.INFO),
		loglib.SetLevel("svr", logging.INFO),
		loglib.SetLevel("tkr", logging.INFO),
		loglib.SetLevel("rec", logging.INFO),
		loglib.SetLevel("mgr", logging.INFO),
		loglib.SetLevel("out", logging.INFO),
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
		repository.SetNodeLimit(10000),
	)
	if err != nil {
		log.Critical("Unable to initialize repository (%v)", err)
		os.Exit(1)
	}

	// writer to publish to file
	wfile, err := processor.NewFileWriter(
		processor.SetLog(logr.GetLog("out")),
		processor.SetSizeLimit(0),
		processor.SetAgeLimit(time.Minute*5),
		processor.SetCompressor(compressor.NewLZ4(
			compressor.SetLog(logr.GetLog("comp")),
		)),
		processor.SetFilePath("logs/"),
	)
	if err != nil {
		log.Critical("Unable to initialize file writer (%v)", err)
		os.Exit(1)
	}

	// writer to publish stuff on zeromq
	wzmq, err := processor.NewZMQWriter(
		processor.SetLog(logr.GetLog("out")),
		processor.SetSocketAddress("127.0.0.1:12345"),
	)
	if err != nil {
		log.Critical("Unable to initialize zeromq writer (%v)", err)
		os.Exit(1)
	}

	// writer to publish stuff to redis
	wredis, err := processor.NewRedisWriter(
		processor.SetLog(logr.GetLog("out")),
		processor.SetServerAddress("127.0.0.1:23456"),
		processor.SetPassword(""),
		processor.SetDatabase(0),
	)
	if err != nil {
		log.Critical("Unable to initialize redis writer (%v)", err)
		os.Exit(1)
	}

	// filter all transactions for zmq output
	ftx, err := processor.NewCommandFilter(
		processor.SetNext(wzmq),
		processor.SetCommands("tx"),
	)
	if err != nil {
		log.Critical("Unable to initialize command filter (%v)", err)
		os.Exit(1)
	}

	// filter some IPs for redis output
	finv, err := processor.NewIPFilter(
		processor.SetNext(wredis),
		processor.SetIPs(
			"208.111.48.35",
			"97.69.174.76",
			"50.181.241.97",
			"173.73.12.206",
			"88.148.169.65",
			"72.11.148.180",
			"195.6.17.142",
			"46.101.168.50",
		),
	)
	if err != nil {
		log.Critical("Unable to initialize IP filter (%v)", err)
		os.Exit(1)
	}

	// filter some address transactions for redis output
	fbase58, err := processor.NewBase58Filter(
		processor.SetNext(wredis),
		processor.SetBase58s(
			"1dice8EMZmqKvrGE4Qc9bUFf9PX3xaYDp",
			"1dice97ECuByXAvqXpaYzSaQuPVvrtmz6",
			"1dice9wcMu5hLF4g81u8nioL5mmSHTApw",
			"1LuckyR1fFHEsXYyx5QK4UFzv3PEAepPMK",
			"1VayNert3x1KzbpzMGt2qdqrAThiRovi8",
		),
	)
	if err != nil {
		log.Critical("Unable to initialize base58 filter (%v)", err)
		os.Exit(1)
	}

	vent, err := processor.NewDummy(
		processor.SetNext(fbase58, finv, ftx, wfile),
	)

	// manager
	mgr, err := manager.New(
		manager.SetLog(logr.GetLog("mgr")),
		manager.SetRepository(repo),
		manager.SetNetwork(wire.MainNet),
		manager.SetVersion(wire.RejectVersion),
		manager.SetConnectionRate(time.Second/25),
		manager.SetInformationRate(time.Second*10),
		manager.SetPeerLimit(1000),
	)
	if err != nil {
		log.Critical("Unable to initialize manager (%v)", err)
		os.Exit(1)
	}

	// server
	svr, err := server.New(
		server.SetLog(logr.GetLog("svr")),
	)
	if err != nil {
		log.Critical("Unable to initialize server (%v)", err)
		os.Exit(1)
	}

	// tracker
	tkr, err := tracker.New(
		tracker.SetLog(logr.GetLog("tkr")),
	)
	if err != nil {
		log.Critical("Unable to initialize tracker (%v)", err)
		os.Exit(1)
	}

	// supervisor
	supervisor, err := supervisor.New(
		supervisor.SetLogger(logr),
		supervisor.SetRepository(repo),
		supervisor.SetManager(mgr),
		supervisor.SetServer(svr),
		supervisor.SetTracker(tkr),
		supervisor.SetProcessor(vent),
	)
	if err != nil {
		log.Critical("Unable to initialize supervisor (%v)", err)
		os.Exit(1)
	}

	_ = supervisor

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
		repo.Close()
		break
	}

	log.Info("[PBTC] All modules shutdown complete")

	os.Exit(1)
}
