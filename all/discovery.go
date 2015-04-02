package all

import (
	"log"
	"net"
	"sync"
)

type discovery struct {
	seedIn    chan string
	addrOut   chan<- string
	waitGroup *sync.WaitGroup
}

func NewDiscovery() *discovery {
	seedIn := make(chan string, bufferDiscoverySeed)

	dsc := &discovery{
		seedIn:    seedIn,
		waitGroup: &sync.WaitGroup{},
	}

	return dsc
}

func (dsc *discovery) GetSeedIn() chan<- string {
	return dsc.seedIn
}

func (dsc *discovery) Start(addrOut chan<- string) {
	dsc.addrOut = addrOut

	dsc.waitGroup.Add(1)
	go dsc.handleSeeds()
}

func (dsc *discovery) Stop() {
	dsc.waitGroup.Wait()
}

func (dsc *discovery) handleSeeds() {
	for seed := range dsc.seedIn {
		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Println("DNS discovery failed:", seed, err)
			continue
		}

		log.Println("Found IPs:", seed, len(ips))
		for _, ip := range ips {
			addr := net.JoinHostPort(ip.String(), protocolPort)
			dsc.addrOut <- addr
		}
	}

	dsc.waitGroup.Done()
}
