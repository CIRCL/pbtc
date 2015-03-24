package all

import (
	"github.com/btcsuite/btcd/wire"
)

const (
	bufferConnector  = 128
	bufferDiscovery  = 128
	bufferMessage    = 128
	bufferRepository = 128
	bufferAcceptor   = 128
	bufferSeeds      = 128
	bufferManager    = 128
)

const (
	protocolVersion = wire.BIP0031Version
	protocolNetwork = wire.TestNet3
	protocolPort    = "18333"
)
