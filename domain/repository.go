package domain

import (
	"net"
)

type Repository interface {
	Count() int
	Bootstrap()
	Save()
	Load()
	Discovered(addr *net.TCPAddr, src *net.TCPAddr)
	Attempted(addr *net.TCPAddr)
	Connected(addr *net.TCPAddr)
	Succeeded(addr *net.TCPAddr)
	Get() *net.TCPAddr
}
