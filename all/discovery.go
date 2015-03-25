package all

import (
	"net"
)

type Discovery struct {
	seedIn  chan string
	addrOut chan<- string
}

func NewDiscovery() *Discovery {
	seedIn := make(chan string, bufferDiscovery)

	dsc := &Discovery{
		seedIn: seedIn,
	}

	return dsc
}

func (dsc *Discovery) GetSeedIn() chan<- string {
	return dsc.seedIn
}

func (dsc *Discovery) Start(addrOut chan<- string) {
	dsc.addrOut = addrOut

	go dsc.handleSeeds()
}

func (dsc *Discovery) Stop() {
	close(dsc.seedIn)
}

func (dsc *Discovery) handleSeeds() {
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
