package adaptor

import (
	"net"
)

// Record defines a common interface for records that describe an event on the
// Bitcoin network. A top-level record will be able to provide the remote
// address and message command that it relates to, while a sub-record only
// provides a string representation of the data.
type Record interface {
	SubRecord
	Address() *net.TCPAddr
	Cmd() string
}

type SubRecord interface {
	String() string
}
