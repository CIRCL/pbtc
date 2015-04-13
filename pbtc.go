package main

import (
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CIRCL/pbtc/all"
	"github.com/btcsuite/btcd/wire"
	"github.com/op/go-logging"
)

const (
	consoleFormat = "%{color}%{time} %{level} %{message}%{color:reset}"
	fileFormat    = "%{time} %{level} %{message}"
)

func main() {
	// initialize backend for console logging
	consoleBackend := logging.NewLogBackend(os.Stderr, "", 0)
	consoleFormatter := logging.MustStringFormatter(consoleFormat)
	consoleFormatted := logging.NewBackendFormatter(consoleBackend, consoleFormatter)
	consoleLeveled := logging.AddModuleLevel(consoleFormatted)
	consoleLeveled.SetLevel(logging.INFO, "pbtc")
	logging.SetBackend(consoleLeveled)

	// set up the logging frontend
	log := logging.MustGetLogger("pbtc")

	// initialize backend for file logging
	file, err := os.Create("pbtc.log")
	if err != nil {
		log.Fatal("Could not create log file")
	}
	defer file.Close()
	fileBackend := logging.NewLogBackend(file, "", 0)
	fileFormatter := logging.MustStringFormatter(fileFormat)
	fileFormatted := logging.NewBackendFormatter(fileBackend, fileFormatter)
	fileLeveled := logging.AddModuleLevel(fileFormatted)
	fileLeveled.SetLevel(logging.DEBUG, "pbtc")
	logging.SetBackend(consoleLeveled, fileLeveled)

	// start program logic
	log.Info("Starting")

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
			log.Info("Stopping")
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

	log.Info("Exiting")

	os.Exit(0)
}
