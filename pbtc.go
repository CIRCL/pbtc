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
	// get main program logger
	logger := all.GetLogHelper("[PBTC]")

	// configure console logging
	agent := all.GetLogAgent()
	agent.AddOutput(log.New(os.Stdout, "", 0), all.LogInfo)

	// configure file logging
	file, err := os.Create("pbtc.log")
	if err != nil {
		logger.Logln(all.LogFatal, "Could not create log file")
		os.Exit(1)
	}
	agent.AddOutput(log.New(file, "", 0), all.LogTrace)
	defer file.Close()

	// start program logic
	logger.Logln(all.LogInfo, "Starting")

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
			logger.Logln(all.LogInfo, "Stopping")
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

	logger.Logln(all.LogInfo, "Exiting")

	os.Exit(0)
}
