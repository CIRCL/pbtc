package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/CIRCL/pbtc/all"
)

func main() {
	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// find all our interfaces & their ips to listen
	ips := all.FindIPs()

	// create list of dns seeds to bootstrap discovery
	seeds := []string{
		//"testnet-seed.alexykot.me",
		"testnet-seed.bitcoin.petertodd.org",
		"testnet-seed.bluematt.me",
		"testnet-seed.bitcoin.schildbach.de",
	}

	// start everything

	// running
	_, _ = fmt.Scanln()

	// stop everything

	return
}
