package repository

import (
	"bytes"
	"net"
)

type ipRange struct {
	start net.IP
	end   net.IP
}

func newIPRange(start string, end string) *ipRange {
	r := &ipRange{
		start: net.ParseIP(start),
		end:   net.ParseIP(end),
	}

	return r
}

func (r *ipRange) includes(ip net.IP) bool {
	if bytes.Compare(r.start, ip) >= 0 && bytes.Compare(r.end, ip) <= 0 {
		return true
	}

	return false
}
