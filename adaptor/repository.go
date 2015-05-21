package adaptor

import (
	"net"
)

// Repository defines a common interface for a node repository. It keeps track
// of all addresses seen on the Bitcoin network and their characteristics. It
// provides clients with a stream of addresses ordered by favourability.
type Repository interface {
	Discovered(addr *net.TCPAddr)
	Attempted(addr *net.TCPAddr)
	Connected(addr *net.TCPAddr)
	Succeeded(addr *net.TCPAddr)
	Retrieve(chan<- *net.TCPAddr)
}
