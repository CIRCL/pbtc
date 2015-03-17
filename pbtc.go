package main

import (
	"log"

	"github.com/CIRCL/pbtc/domain"
)

func main() {
	log.Println("Passive Bitcoin by CIRL")

	// find all valid local IPv4 addresses
	ips := domain.FindIPs()

	// initialize our listening node on those adresses
	server, err := domain.NewServer(ips)
	if err != nil {
		log.Println(err)
	}

	_ = server
}
