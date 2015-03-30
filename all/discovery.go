package all

import (
	"log"
	"net"
)

type discovery struct {
	seedIn  chan string
	addrOut chan<- string
}

func NewDiscovery() *discovery {
	seedIn := make(chan string, bufferDiscovery)

	dsc := &discovery{
		seedIn: seedIn,
	}

	return dsc
}

func (dsc *discovery) GetSeedIn() chan<- string {
	return dsc.seedIn
}

func (dsc *discovery) Start(addrOut chan<- string) {
	dsc.addrOut = addrOut

	go dsc.handleSeeds()
}

func (dsc *discovery) Stop() {
	close(dsc.seedIn)
}

func (dsc *discovery) handleSeeds() {
	for seed := range dsc.seedIn {

		log.Println("Running DNS discovery:", seed)

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
}
