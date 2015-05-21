package adaptor

import (
	"github.com/btcsuite/btcd/wire"
)

// Manager defines a common interface for peer management. It can be used to
// keep track of peer state and a global shared state.
type Manager interface {
	Connected(Peer)
	Ready(Peer)
	Stopped(Peer)
	Knows(wire.ShaHash) bool
	Mark(wire.ShaHash)
}
