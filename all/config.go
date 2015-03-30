package all

import (
	"time"

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
	ProtocolVersion = wire.BIP0031Version
	ProtocolNetwork = wire.TestNet3
	protocolPort    = "18333"
)

const (
	maxConnsPerSec = 4
	maxConnsTotal  = 8192
	maxNodesTotal  = 32768
)

const (
	timeoutRecv = 1 * time.Second
	timeoutSend = 1 * time.Second
	timeoutDial = 4 * time.Second
)

const (
	backoffInitial    = 60
	backoffMaximum    = 60 * 60
	backoffMultiplier = 2
	backoffRandomizer = 0.5
)
