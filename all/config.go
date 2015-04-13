package all

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	bufferManagerNew  = 1 // how many new peers can be queued for adding to the manager
	bufferManagerDone = 1 // how many stopped peers can be queued for removal from manager
	bufferPeerRecv    = 1 // the amount of messages that can be queued after reception
	bufferPeerSend    = 1 // the amount of messages that can be queued for sending
	bufferRepoAddr    = 1 // how many addresses can be queued for updating in repository
	bufferRepoNode    = 1 // how many new nodes can be queued for adding to repository
)

const (
	protocolNetwork = wire.TestNet3      // what bitcoin network do we connect to
	protocolVersion = wire.RejectVersion // what protocol version do we try to use
)

const (
	maxConnsPerSec  = 4     // maximum outgoing tcp connections per second
	maxAddrAttempts = 128   // maximum times we try to get a good address to connect to
	maxPeerCount    = 16    // maximum number of concurrent connected peers
	maxNodeCount    = 32768 // maximum number of records in node repository
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

const (
	nodeSaveInterval = time.Minute * 1 // interval at which we save the node index to file
)
