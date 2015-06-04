package adaptor

import (
	"net"
)

// Peer defines a common interface for managers to communicate with peers. It
// can be used to treat various peers differently.
type Peer interface {
	String() string
	Addr() *net.TCPAddr
	Connect()
	Start()
	Stop()
	Greet()
	Poll()
}
