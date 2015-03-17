package domain

import (
	"net"
)

type PeerRepository interface {
	Store(peer Peer)
	FindByIp(ip net.IP) Peer
}

type Peer struct {
	ip net.IP
}

func newPeer(ip net.IP) (*Peer, error) {

	peer := &Peer{
		ip: ip,
	}

	return peer, nil
}

func (peer *Peer) connect() {

}

func (peer *Peer) disconnect() {

}

func (peer *Peer) send() {

}

func (peer *Peer) receive() {

}
