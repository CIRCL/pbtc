package all

import (
	"log"
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

	log.Println("Starting discovery agent")

	dAgent.addrOut = addrOut

	go dAgent.handleSeeds()
}

func (dAgent *discoveryAgent) Stop() {

	log.Println("Stopping discovery agent")

	close(dAgent.seedIn)
}

func (dAgent *discoveryAgent) handleSeeds() {

	for seed := range dAgent.seedIn {

		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Println("DIscovery failed:", seed)
			continue
		}

		for _, ip := range ips {
			log.Println("IP discovered from seed:", ip)

			addr := net.JoinHostPort(ip.String(), protocolPort)
			dAgent.addrOut <- addr
		}
	}
}
