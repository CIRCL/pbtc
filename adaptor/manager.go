package adaptor

import (
	"github.com/btcsuite/btcd/wire"
)

type Manager interface {
	Connected(Peer)
	Ready(Peer)
	Stopped(Peer)
	Knows(wire.ShaHash) bool
	Mark(wire.ShaHash)
}
