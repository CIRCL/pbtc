package main

import (
	"log"
	"net"
	"strconv"
	"time"

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

	// placeholder to stay compilable
	_ = server

	// empty list of IPs of nodes
	var nodes []net.IP

	// list of dns seeds for bootstrapping
	seeds := []string{
		"seed.bitcoin.sipa.be",
		"dnsseed.bluematt.me",
		"dnsseed.bitcoin.dashjr.org",
		"seed.bitcoinstats.com",
		"bitseed.xf2.org"}

	// discover IPs for given dns seeds
	for _, seed := range seeds {
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

			log.Println("Discovered at " + ip.String())

			nodes = append(nodes, ip)
		}
	}

	// list of connections
	var conns []net.Conn

	// connect to all discovered nodes
	for _, node := range nodes {
		log.Println("Connecting to " + node.String())

		conn, err := net.DialTimeout("tcp4", node.String()+":8333", 1*time.Second)
		if err != nil {
			log.Println(err)
			continue
		}

		conns = append(conns, conn)
	}

	log.Println(strconv.Itoa(len(conns)) + " total connections")

	// wait for a bit
	timer := time.NewTimer(8 * time.Second)
	<-timer.C

	// close all connections
	for _, conn := range conns {
		log.Println("Disconnecting from " + conn.RemoteAddr().String())

		conn.Close()
	}

	log.Println("Exiting")
}
