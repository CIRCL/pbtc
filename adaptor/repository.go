package adaptor

import (
	"net"
)

type Repository interface {
	Discovered(addr *net.TCPAddr)
	Attempted(addr *net.TCPAddr)
	Connected(addr *net.TCPAddr)
	Succeeded(addr *net.TCPAddr)
	Retrieve() *net.TCPAddr
}
