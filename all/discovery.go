package all

import (
	"net"
)

type discoveryAgent struct {
	seedIn  chan string
	addrOut chan<- string
}

func NewDiscoveryAgent() *discoveryAgent {

	seedIn := make(chan string, bufferDiscovery)

	dAgent := &discoveryAgent{
		seedIn: seedIn,
	}

	return dAgent
}

func (dAgent *discoveryAgent) GetSeedIn() chan<- string {
	return dAgent.seedIn
}

func (dAgent *discoveryAgent) Start(addrOut chan<- string) {

	dAgent.addrOut = addrOut

	go dAgent.handleSeeds()
}

func (dAgent *discoveryAgent) Stop() {

}

func (dAgent *discoveryAgent) handleSeeds() {

	for seed := range dAgent.seedIn {

		ips, err := net.LookupIP(seed)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			addr := net.JoinHostPort(ip.String(), protocolPort)
			dAgent.addrOut <- addr
		}
	}
}
