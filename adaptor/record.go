package adaptor

import (
	"net"
	"time"
)

// Record defines a common interface for records that describe an event on the
// Bitcoin network. A top-level record will be able to provide the remote
// address and message command that it relates to, while a sub-record only
// provides a string representation of the data.
type Record interface {
	Timestamp() time.Time
	RemoteAddress() *net.TCPAddr
	LocalAddress() *net.TCPAddr
	Command() string
	String() string
}
