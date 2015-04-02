package all

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type discovery struct {
	seedIn    chan string
	addrOut   chan<- string
	sigSeed   chan struct{}
	waitGroup *sync.WaitGroup
	state     uint32
}

func NewDiscovery() *discovery {
	dsc := &discovery{
		seedIn:    make(chan string, bufferDiscoverySeed),
		waitGroup: &sync.WaitGroup{},
		state:     stateIdle,
	}

	return dsc
}

func (dsc *discovery) GetSeedIn() chan<- string {
	return dsc.seedIn
}

func (dsc *discovery) Start(addrOut chan<- string) {
	if !atomic.CompareAndSwapUint32(&dsc.state, stateIdle, stateRunning) {
		return
	}

	log.Println("[DSC] Starting")

	dsc.sigSeed = make(chan struct{}, 1)

	dsc.addrOut = addrOut

	dsc.waitGroup.Add(1)
	go dsc.handleSeeds()

	log.Println("[DSC] Started")
}

func (dsc *discovery) Stop() {
	if !atomic.CompareAndSwapUint32(&dsc.state, stateRunning, stateIdle) {
		return
	}

	log.Println("[DSC] Stopping")

	close(dsc.sigSeed)

	dsc.waitGroup.Wait()

	log.Println("[DSC] Stopped")
}

func (dsc *discovery) handleSeeds() {
SeedLoop:
	for {
		select {
		case _, ok := <-dsc.sigSeed:
			if !ok {
				break SeedLoop
			}

		case seed, ok := <-dsc.seedIn:
			if !ok {
				break SeedLoop
			}

			ips, err := net.LookupIP(seed)
			if err != nil {
				log.Println("[DSC] DNS discovery failed:", seed, err)
				continue SeedLoop
			}

			log.Println("[DSC] Found IPs:", seed, len(ips))
			for _, ip := range ips {
				addr := net.JoinHostPort(ip.String(), protocolPort)
				dsc.addrOut <- addr
			}
		}
	}

	dsc.waitGroup.Done()
}
