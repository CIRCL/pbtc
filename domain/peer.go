package domain

import (
	"net"
)

type ConnectionRepository interface {
	Store(peer Peer)
	FindByIP(ip net.IP) Peer
}

type Peer struct {
	ip net.IP
}
