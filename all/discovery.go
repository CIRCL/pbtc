package all

import (
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

		ips, err := net.LookupIP(seed)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			addr := net.JoinHostPort(ip.String(), protocolPort)
			dsc.addrOut <- addr
		}
	}
}
