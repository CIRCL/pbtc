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

// Filter is responsible for writing records to a file. It can filter events
// to only show certain types, or limit them to certain IP/Bitcoin addresses.
// It will periodically rotate the files and supports compression.
type Filter struct {
	wg         *sync.WaitGroup
	cmdConfig  map[string]bool
	ipConfig   map[string]bool
	addrConfig map[string]bool
	txIndex    *parmap.ParMap
	blockIndex *parmap.ParMap
	writers    []adaptor.Writer

	log  adaptor.Log
	comp adaptor.Compressor

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration
}

// New creates a new filter with the given options.
func New(options ...func(*Filter)) (*Filter, error) {
	rec := &Filter{
		wg:         &sync.WaitGroup{},
		cmdConfig:  make(map[string]bool),
		ipConfig:   make(map[string]bool),
		addrConfig: make(map[string]bool),
		txIndex:    parmap.New(),
		blockIndex: parmap.New(),
		writers:    make([]adaptor.Writer, 0, 2),
	}

	for _, option := range options {
		option(rec)
	}

	return rec, nil
}

// SetLogger injects the logger to be used for logging.
func SetLog(log adaptor.Log) func(*Filter) {
	return func(rec *Filter) {
		rec.log = log
	}
}

// SetTypes sets the type of events to write to file.
func FilterTypes(cmds ...string) func(*Filter) {
	return func(rec *Filter) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

func FilterIPs(ips ...string) func(*Filter) {
	return func(rec *Filter) {
		for _, ip := range ips {
			rec.ipConfig[ip] = true
		}
	}
}

func FilterAddresses(addrs ...string) func(*Filter) {
	return func(rec *Filter) {
		for _, addr := range addrs {
			rec.addrConfig[addr] = true
		}
	}
}

func AddWriter(w adaptor.Writer) func(*Filter) {
	return func(rec *Filter) {
		rec.writers = append(rec.writers, w)
	}
}

// Message will process a given message and log it if it's elligible.
func (rec *Filter) Message(msg wire.Message, ra *net.TCPAddr,
	la *net.TCPAddr) {
	if len(rec.cmdConfig) > 0 {
		if !rec.cmdConfig[msg.Command()] {
			return
		}
	}

	if len(rec.ipConfig) > 0 {
		if !rec.ipConfig[ra.IP.String()] {
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
		if rec.blockIndex.Has(m.BlockSha()) {
			return
		}

		rec.blockIndex.Insert(m.BlockSha())
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
		if rec.txIndex.Has(m.TxSha()) {
			return
		}

		rec.txIndex.Insert(m.TxSha())
		tx := records.NewTransactionRecord(m, ra, la)
		ok := true

		if len(rec.addrConfig) > 0 {
			ok = false
		Outer:
			for addr := range rec.addrConfig {
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

	for _, writer := range rec.writers {
		writer.Line(record.String())
	}
}
