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
	wredis, err := writer.NewRedis(
		writer.SetLogRedis(logr.GetLog("out")),
		writer.SetAddressRedis("127.0.0.1:23456"),
		writer.SetPassword(""),
		writer.SetDatabase(0),
	)
	if err != nil {
		log.Critical("Unable to initialize redis writer (%v)", err)
		os.Exit(3)
	}

	// recorder that doesn't filter
	rec_all, err := filter.New(
		filter.SetLog(logr.GetLog("rec")),
		filter.AddWriter(wfile),
		filter.AddWriter(wzmq),
		filter.AddWriter(wredis),
	)
	if err != nil {
		log.Critical("Unable to initialize full filter (%v)", err)
		os.Exit(4)
	}

	// filter filtering only transactions to given addresses
	rec_addr, err := filter.New(
		filter.SetLog(logr.GetLog("rec")),
		filter.FilterTypes(wire.CmdTx),
		filter.FilterAddresses(
			"1dice8EMZmqKvrGE4Qc9bUFf9PX3xaYDp",
			"1dice97ECuByXAvqXpaYzSaQuPVvrtmz6",
			"1dice9wcMu5hLF4g81u8nioL5mmSHTApw",
			"1dice7fUkz5h4z2wPc1wLMPWgB5mDwKDx",
			"1dice7W2AicHosf5EL3GFDUVga7TgtPFn",
			"1dice6YgEVBf88erBFra9BHf6ZMoyvG88",
			"1diceDCd27Cc22HV3qPNZKwGnZ8QwhLTc",
			"1NxaBCFQwejSZbQfWcYNwgqML5wWoE3rK4",
			"1LuckyR1fFHEsXYyx5QK4UFzv3PEAepPMK",
			"1VayNert3x1KzbpzMGt2qdqrAThiRovi8",
		),
		filter.AddWriter(wfile),
	)
	if err != nil {
		log.Critical("Unable to initialize address filter (%v)", err)
		os.Exit(4)
	}

	// filter to monitor a set of ip addresses
	rec_ip, err := filter.New(
		filter.SetLog(logr.GetLog("rec")),
		filter.FilterTypes(wire.CmdInv, wire.CmdPing, wire.CmdVersion),
		filter.FilterIPs(
			"208.111.48.35",
			"97.69.174.76",
			"50.181.241.97",
			"173.73.12.206",
			"88.148.169.65",
			"72.11.148.180",
			"195.6.17.142",
			"46.101.168.50",
		),
		filter.AddWriter(wfile),
	)
	if err != nil {
		log.Critical("Unable to initialize ip filter recorder (%v)", err)
		os.Exit(4)
	}

	// manager
	mgr, err := manager.New(
		manager.SetLog(logr.GetLog("mgr")),
		manager.SetPeerLog(logr.GetLog("peer")),
		manager.SetRepository(repo),
		manager.AddRecorder(rec_all),
		manager.AddRecorder(rec_addr),
		manager.AddRecorder(rec_ip),
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
		mgr.Stop()
		repo.Stop()
		wredis.Stop()
		wzmq.Stop()
		wfile.Stop()
		logr.Stop()
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
