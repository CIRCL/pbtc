package all

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	bufferManagerNew  = 1 // new peers queue in manager
	bufferManagerDone = 1 // done peers queue in manager
	bufferPeerRecv    = 1 // message reception queue for peers
	bufferPeerSend    = 1 // message expedition queue for peers
	bufferRepoAddr    = 1 // address update queue for repository
	bufferRepoNode    = 1 // node addition queue for repository
)

const (
	protocolNetwork = wire.TestNet3      // bitcoin network to connect to
	protocolVersion = wire.RejectVersion // maximum protocol version for peers
)

const (
	maxConnsPerSec = 4     // maximum outgoing tcp connections per second
	maxPeerCount   = 1024  // maximum number of concurrent connected peers
	maxNodeCount   = 32768 // maximum number of records in node repository
)

const (
	timeoutRecv = 1 * time.Second // timeout on receives before rechecking
	timeoutSend = 1 * time.Second // timeout on sends before discarding message
	timeoutDial = 4 * time.Second // timeout before considering a dial failed
	timeoutIdle = 5 * time.Minute // timeout for peers when no messages are received
	timeoutPing = 2 * time.Minute // timeout before sending a ping for keep alive
)

const (
	backoffInitial    = 60 * time.Second // initial time before retrying a node
	backoffMaximum    = 60 * time.Minute // maximum time before retrying a node
	backoffMultiplier = 1.99             // increase factor on each retry
	backoffRandomizer = 0.27             // random factor on each retry
)

const (
	logLimitSize = 1024 * 1024 * 64 // maximum log file size before rotating (bytes)
	logLimitTime = 5 * time.Minute  // maximum time for one log file before rotating
	logInfoTick  = 1 * time.Second  // time between printing log info to console
)

const (
	userAgentName    = "satoshi" // user agent that we identify to peers with
	userAgentVersion = "0.8.2"   // user agent version we indicate to peers
)
