package adaptor

import (
	"net"
)

type Peer interface {
	String() string
	Addr() *net.TCPAddr
	Cleanup()
	Poll()
}
