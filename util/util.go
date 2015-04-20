package util

import (
	"net"
)

// FindLocalIPs finds all IPs associated with local interfaces.
func FindLocalIPs() ([]net.IP, error) {
	// create empty slice of ips to return
	var ips []net.IP

	// get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var last_err error

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
			last_err = err
		}

		// iterate through addresses to get valid ips
		for _, addr := range addrs {
			// get the IP for valid IP types
			var ip net.IP
			switch t := addr.(type) {
			case *net.IPNet:
				ip = t.IP
			case *net.IPAddr:
				ip = t.IP
			default:
				continue
			}

			// if the IP is a loopback IP, skip
			if ip.IsLoopback() {
				continue
			}

			// if the IP is not a valid IPv4 address, skip
			ipv4 := ip.To4()
			if ipv4 == nil {
				continue
			}

			// append the IP to the slice of valid IPs
			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		return nil, last_err
	}

	// return the slice of valid IPs, can be zero length and empty
	return ips, nil
}

// MinUint32 returns the smaller of two uint32. It is used as a shortcut
// to negotiate the version number with new peers.
func MinUint32(x uint32, y uint32) uint32 {
	if x > y {
		return x
	}

	return y
}
