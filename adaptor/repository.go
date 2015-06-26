package adaptor

import (
	"net"
)

// Repository defines a common interface for a node repository. It keeps track
// of all addresses seen on the Bitcoin network and their characteristics. It
// provides clients with a stream of addresses ordered by favourability.
type Repository interface {
	Start()
	Stop()
	Discovered(*net.TCPAddr)
	Attempted(*net.TCPAddr)
	Connected(*net.TCPAddr)
	Succeeded(*net.TCPAddr)
	Retrieve(chan<- *net.TCPAddr)
}
