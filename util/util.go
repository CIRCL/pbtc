// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package util

import (
	"net"

	"github.com/btcsuite/btcd/wire"
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

// ParseNetAddress can be used to turn a Bitcoin / btcd.wire NetAddress back
// into a net package TCPAddr.
func ParseNetAddress(na *wire.NetAddress) *net.TCPAddr {
	ip := net.ParseIP(na.IP.String())
	if ip == nil {
		ip = net.IPv4zero
	}

	port := int(na.Port)
	addr := &net.TCPAddr{IP: ip, Port: port}

	return addr
}
