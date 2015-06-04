package adaptor

import (
	"github.com/btcsuite/btcd/wire"
)

// Manager defines the interface used by peers to communicate with their
// manager. It is notified of peer state, keeps track of shared state and
// decides on actions depending on state. Different managers can implement
// different behaviours.
type Manager interface {
	Connected(Peer)
	Ready(Peer)
	Stopped(Peer)
	Knows(wire.ShaHash) bool
	Mark(wire.ShaHash)
}
