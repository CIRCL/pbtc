package adaptor

import (
	"net"

	"github.com/btcsuite/btcd/wire"
)

// Filter defines an interface for filters to work on messages from the Bitcoin
// network. It will filter the messages according to a number of criteria
// before forwarding them to the added writers.
type Filter interface {
	Message(msg wire.Message, la *net.TCPAddr, ra *net.TCPAddr)
}
