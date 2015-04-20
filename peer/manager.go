package peer

import (
	"github.com/btcsuite/btcd/wire"
)

type Manager interface {
	Started(peer *Peer)
	Ready(peer *Peer)
	Stopped(peer *Peer)
	Message(msg wire.Message)
}
