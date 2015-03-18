package usecases

import (
	"net"
)

type ConnectionRepository interface {
	Store(peer Peer)
	FindByIP(ip net.IP) Peer
}

type Peer struct {
	ip   net.IP
	conn net.Conn
}

func (*Peer) Close() {

}
