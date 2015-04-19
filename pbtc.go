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

	"github.com/CIRCL/pbtc/application"
)

const (
	consoleFormat = "%{color}%{time} %{level} %{shortfile} %{message}%{color:reset}"
	fileFormat    = "%{time} %{level} %{shortfile} %{message}"
)

func main() {
	// initialize backend for console logging
	consoleBackend := logging.NewLogBackend(os.Stderr, "", 0)
	consoleFormatter := logging.MustStringFormatter(consoleFormat)
	consoleFormatted := logging.NewBackendFormatter(consoleBackend, consoleFormatter)
	consoleLeveled := logging.AddModuleLevel(consoleFormatted)
	consoleLeveled.SetLevel(logging.DEBUG, "pbtc")
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
	log.Info("PBTC starting")

	// catch signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// initialize our modules
	mon := application.NewMonitor()

	// start our modules
	mon.Start(wire.TestNet3, wire.RejectVersion)

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case os.Interrupt:
			log.Info("PBTC stopping")
			break SigLoop

		case syscall.SIGTERM:

		case syscall.SIGHUP:

		case syscall.SIGINT:

		case syscall.SIGQUIT:
		}
	}

	// stop our modules
	mon.Stop()

	log.Info("PBTC stopped")

	os.Exit(0)
}
