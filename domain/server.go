package domain

import (
	"net"
)

type Server struct {
}

func newServer(ips []net.IP) (*Server, error) {

	server := &Server{}

	return server, nil
}

func findIPs() []net.IP {
	// create empty slice of ips to return
	ips := make([]net.IP, 0, 8)

	// get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		// logger
		return ips
	}

	// iterate through interfaces to find valid ips
	for _, iface := range ifaces {

		// if the interface is down, skip
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// if the interface is loopback, skip
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// get all interface addresses
		addrs, err := iface.Addrs()
		if err != nil {
			// logger
			continue
		}

		// iterate through addresses to get valid ips
		for _, addr := range addrs {

			// get the IP for valid IP types
			switch t := addr.(type) {
			case *net.IPNet:
				ip := t.IP
			case *net.IPAddr:
				ip := t.IP
			default:
				continue
			}

			// if the IP is a loopback IP, skip
			if ip.IsLoopback() {
				continue
			}

			// if the IP is not a valid IPv4 address, skip
			ip = ip.To4()
			if ip == nil {
				continue
			}

			// append the IP to the slice of valid IPs
			ips = append(ips, ip)
		}
	}

	// return the slice of valid IPs, can be zero length and empty
	return ips
}
