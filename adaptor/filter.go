package adaptor

import (
	"net"

	"github.com/btcsuite/btcd/wire"
)

// Recorder defines a common interface for a recorder that can record events
// and messages on the Bitcoin network. It can be used to store events in
// different formats and on different media.
type Filter interface {
	Message(msg wire.Message, la *net.TCPAddr, ra *net.TCPAddr)
}
