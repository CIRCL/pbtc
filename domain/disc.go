package domain

import (
	"log"
	"net"
)

type Discoverer struct {
	seedIn  chan string
	nodeOut chan string
}

func NewDiscoverer() (*Discoverer, error) {
	disc := &Discoverer{}

	return disc, nil
}

func (disc *Discoverer) Start(seedIn chan string, nodeOut chan string) {
	disc.seedIn = seedIn
	disc.nodeOut = nodeOut

	go disc.handleSeeds()
}

func (disc *Discoverer) Stop() {
	close(disc.seedIn)
}

func (disc *Discoverer) handleSeeds() {
	for seed := range disc.seedIn {
		ips, err := net.LookupIP(seed)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, ip := range ips {
			ip := ip.To4()
			if ip == nil {
				continue
			}

			log.Println("Discovered node at", ip.String())
			disc.nodeOut <- ip.String()
		}
	}
}

func (disc *Discoverer) AddSeed(seed string) {
	disc.seedIn <- seed
}
