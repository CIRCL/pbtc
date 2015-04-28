package adaptor

import (
	"net"
)

type Peer interface {
	String() string
	Addr() *net.TCPAddr
	Pending() bool
	Connected() bool
	Ready() bool
	Connect()
	Start()
	Greet()
	Stop()
	Poll()
}
