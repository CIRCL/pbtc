package domain

import (
	"log"
	"net"
)

type Discoverer struct {
	seedIn  <-chan string
	nodeOut chan<- string
}

func NewDiscoverer(seedIn <-chan string, nodeOut chan<- string) (*Discoverer, error) {
	disc := &Discoverer{
		seedIn:  seedIn,
		nodeOut: nodeOut,
	}

	return disc, nil
}

func (disc *Discoverer) Start() {
	go func() {
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

				disc.nodeOut <- ip.String()
			}
		}

		close(disc.nodeOut)
	}()
}
