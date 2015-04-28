package adaptor

import (
	"net"

	"github.com/btcsuite/btcd/wire"
)

type Recorder interface {
	Message(msg wire.Message, la *net.TCPAddr, ra *net.TCPAddr)
}
