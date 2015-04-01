package all

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	bufferManagerAddress    = 1024
	bufferManagerPeer       = 128
	bufferManagerConnection = 64
	bufferManagerEvent      = 1024

	bufferDiscoverySeed = 16
)

const (
	protocolVersion = wire.BIP0031Version
	protocolNetwork = wire.TestNet3
	protocolPort    = "18333"
)

const (
	maxConnsPerSec = 4
	maxConnsTotal  = 8192
	maxNodesTotal  = 32768
)

const (
	timeoutRecv = 4 * time.Second
	timeoutSend = 4 * time.Second
	timeoutDial = 4 * time.Second
)

const (
	backoffInitial    = 60 * time.Second
	backoffMaximum    = 60 * time.Minute
	backoffMultiplier = 1.99
	backoffRandomizer = 0.27
)

const (
	logLimitSize = 1024 * 1024 * 64
	logLimitTime = 5 * time.Minute
)
