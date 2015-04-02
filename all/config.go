package all

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	bufferManagerAddress    = 1
	bufferManagerPeer       = 1
	bufferManagerConnection = 1
	bufferManagerEvent      = 1

	bufferDiscoverySeed = 1

	bufferServerAddress = 1

	bufferPeerRecv = 1
	bufferPeerSend = 1
)

const (
	protocolVersion = wire.BIP0031Version
	protocolNetwork = wire.TestNet3
	protocolPort    = "18333"
)

const (
	maxConnsPerSec = 4
	maxConnsTotal  = 1024
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
	logInfoTick  = 1 * time.Second
)
