package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/supervisor"
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

	cfg := &Config{}
	err := gcfg.ReadFileInto(cfg, "pbtc.cfg")
	if err != nil {
		fmt.Println("Could not read config file (%v)", err)
		os.Exit(1)
	}

	logr, err := logger.New()
	if err != nil {
		fmt.Println("Logger initialization failed (%v)", err)
		os.Exit(1)
	}

	log := logr.GetLog("main")
	log.Info("PBTC initializing...")

	// initialize supervisor
	supervisor, err := supervisor.New(logr)
	if err != nil {
		log.Critical("Supervisor initialization failed (%v)", err)
		os.Exit(1)
	}

	log.Info("PBTC initialization complete")

	// start supervisor
	log.Info("Starting modules")
	supervisor.Start()

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case syscall.SIGINT:
			log.Info("PBTC shutting down...")
			break SigLoop

		case syscall.SIGHUP:
			continue
		}
	}

	// we will initialize shutdown in a non-blocking way
	c := make(chan struct{})
	go func() {
		log.Info("Stopping modules")
		supervisor.Stop()
		c <- struct{}{}
	}()

	// if the shutdown completes, we simple quit normally
	// however, if we receive another signal during shutdown, we panic
	// this allows us to see the stacktrace in case shutdown blocks somewhere
	select {
	case <-sigc:
		panic("SHUTDOWN FAILED")

	case <-c:
		log.Info("Modules stopped")
		break
	}

	log.Info("PBTC shutdown complete")
	os.Exit(0)
}
