package filter

import (
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
	"github.com/CIRCL/pbtc/records"
)

// Filter is the standard implementation of the filter adaptor. It takes
// messages from the Bitcoin network as input and filters them according to
// certain criteria. After the filtering pipeline, it will forward the
// remaining messages to the added writers.
type Filter struct {
	log  adaptor.Log
	comp adaptor.Compressor

	wg         *sync.WaitGroup
	cmdConfig  map[string]bool
	ipConfig   map[string]bool
	addrConfig map[string]bool
	txIndex    *parmap.ParMap
	blockIndex *parmap.ParMap
	writers    []adaptor.Writer

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration
}

// New creates a new filter with the given options. The options are provided
// as a parameter list of package functions, allowing us to define the various
// aspects of the filter.
func New(options ...func(*Filter)) (*Filter, error) {
	filter := &Filter{
		wg:         &sync.WaitGroup{},
		cmdConfig:  make(map[string]bool),
		ipConfig:   make(map[string]bool),
		addrConfig: make(map[string]bool),
		txIndex:    parmap.New(),
		blockIndex: parmap.New(),
		writers:    make([]adaptor.Writer, 0, 2),
	}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

// SetLogger injects the logger to be used for logging.
func SetLog(log adaptor.Log) func(*Filter) {
	return func(filter *Filter) {
		filter.log = log
	}
}

// FilterTypes defines a filter on the type of message. If defined, only
// messages of the given types will be forwarded.
func FilterTypes(cmds ...string) func(*Filter) {
	return func(filter *Filter) {
		for _, cmd := range cmds {
			filter.cmdConfig[cmd] = true
		}
	}
}

// FilterIPs defines a filter on the remote IP address of messages. If provided,
// only messages from the given IPs will be forwarded.
func FilterIPs(ips ...string) func(*Filter) {
	return func(filter *Filter) {
		for _, ip := range ips {
			filter.ipConfig[ip] = true
		}
	}
}

// FilterAddresses deines a filter on the Bitcoin address of messages. If
// provided, only transactions that have any of the given Bitcoin addresses as
// an output will be forwarded. All other messages will still be forwarded, so
// be sure to use FilterTypes to filter only for transactions if this is the
// desired behaviour.
func FilterAddresses(addrs ...string) func(*Filter) {
	return func(filter *Filter) {
		for _, addr := range addrs {
			filter.addrConfig[addr] = true
		}
	}
}

// AddWriter adds an output channel for the filtered messages to the filter. The
// messages will be forwarded to every writer added to the filter.
func AddWriter(w adaptor.Writer) func(*Filter) {
	return func(filter *Filter) {
		filter.writers = append(filter.writers, w)
	}
}

// Init initializes the filter.
func (filter *Filter) Init() {
}

// Close shuts the filter down.
func (filter *Filter) Close() {
}

// Message submits a new message to the filter and will send it through the
// filtering pipeline. After passing the filters, we convert the message to
// a record which can be handled by the added writers.
func (filter *Filter) Message(msg wire.Message, ra *net.TCPAddr,
	la *net.TCPAddr) {
	if len(filter.cmdConfig) > 0 {
		if !filter.cmdConfig[msg.Command()] {
			return
		}
	}

	if len(filter.ipConfig) > 0 {
		if !filter.ipConfig[ra.IP.String()] {
			return
		}
	}

	var record adaptor.Record

	switch m := msg.(type) {
	case *wire.MsgAddr:
		record = records.NewAddressRecord(m, ra, la)

	case *wire.MsgAlert:
		record = records.NewAlertRecord(m, ra, la)

	case *wire.MsgBlock:
		if filter.blockIndex.Has(m.BlockSha()) {
			return
		}

		filter.blockIndex.Insert(m.BlockSha())
		record = records.NewBlockRecord(m, ra, la)

	case *wire.MsgHeaders:
		record = records.NewHeadersRecord(m, ra, la)

	case *wire.MsgInv:
		record = records.NewInventoryRecord(m, ra, la)

	case *wire.MsgPing:
		record = records.NewPingRecord(m, ra, la)

	case *wire.MsgPong:
		record = records.NewPongRecord(m, ra, la)

	case *wire.MsgReject:
		record = records.NewRejectRecord(m, ra, la)

	case *wire.MsgVersion:
		record = records.NewVersionRecord(m, ra, la)

	case *wire.MsgTx:
		if filter.txIndex.Has(m.TxSha()) {
			return
		}

		filter.txIndex.Insert(m.TxSha())
		tx := records.NewTransactionRecord(m, ra, la)
		ok := true

		if len(filter.addrConfig) > 0 {
			ok = false
		Outer:
			for addr := range filter.addrConfig {
				if tx.HasAddress(addr) {
					ok = true
					break Outer
				}
			}
		}

		if !ok {
			return
		}

		record = tx

	case *wire.MsgFilterAdd:
		record = records.NewFilterAddRecord(m, ra, la)

	case *wire.MsgFilterClear:
		record = records.NewFilterClearRecord(m, ra, la)

	case *wire.MsgFilterLoad:
		record = records.NewFilterLoadRecord(m, ra, la)

	case *wire.MsgGetAddr:
		record = records.NewGetAddrRecord(m, ra, la)

	case *wire.MsgGetBlocks:
		record = records.NewGetBlocksRecord(m, ra, la)

	case *wire.MsgGetData:
		record = records.NewGetDataRecord(m, ra, la)

	case *wire.MsgGetHeaders:
		record = records.NewGetHeadersRecord(m, ra, la)

	case *wire.MsgMemPool:
		record = records.NewMemPoolRecord(m, ra, la)

	case *wire.MsgMerkleBlock:
		record = records.NewMerkleBlockRecord(m, ra, la)

	case *wire.MsgNotFound:
		record = records.NewNotFoundRecord(m, ra, la)

	case *wire.MsgVerAck:
		record = records.NewVerAckRecord(m, ra, la)
	}

	for _, writer := range filter.writers {
		writer.Line(record.String())
	}
}
