package adaptor

import (
	"net"
)

// Record defines a common interface for records that describe an event on the
// Bitcoin network. They provide the output in string and binary format at this
// point.
type Record interface {
	SubRecord
	Address() *net.TCPAddr
	Cmd() string
}

type SubRecord interface {
	String() string
}
