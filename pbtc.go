// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CIRCL/pbtc/supervisor"
)

func main() {
	fmt.Println("Copyright (c) 2015 Max Wolter")
	fmt.Println("Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg")
	fmt.Println("                          (c/o smile, security made in Lëtzebuerg, Groupement")
	fmt.Println("                          d'Intérêt Economique)")
	fmt.Println("")
	fmt.Println("PBTC is free software: you can redistribute it and/or modify")
	fmt.Println("it under the terms of the GNU Affero General Public License as published by")
	fmt.Println("the Free Software Foundation, either version 3 of the License, or")
	fmt.Println("(at your option) any later version.")
	fmt.Println("")
	fmt.Println("PBTC is distributed in the hope that it will be useful,")
	fmt.Println("but WITHOUT ANY WARRANTY; without even the implied warranty of")
	fmt.Println("MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the")
	fmt.Println("GNU Affero General Public License for more details.")
	fmt.Println("")
	fmt.Println("You should have received a copy of the GNU Affero General Public License")
	fmt.Println("along with PBTC.  If not, see <http://www.gnu.org/licenses/>.")
	fmt.Println("")

	// catch signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	signal.Notify(sigc, syscall.SIGHUP)

	// use all cpu cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// seed the random generator
	rand.Seed(time.Now().UnixNano())

	// initialize supervisor
	supervisor, err := supervisor.New()
	if err != nil {
		fmt.Printf("Initialization failed (%v)\n", err)
		os.Exit(1)
	}

	// start supervisor
	supervisor.Start()

	// wait for signals in blocking loop
SigLoop:
	for sig := range sigc {
		switch sig {
		case syscall.SIGINT:
			break SigLoop

		case syscall.SIGHUP:
			continue
		}
	}

	// we will initialize shutdown in a non-blocking way
	c := make(chan struct{})
	go func() {
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
		break
	}

	os.Exit(0)
}
