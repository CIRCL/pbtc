package recorder

import (
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
)

// Recorder is responsible for writing records to a file. It can filter events
// to only show certain types, or limit them to certain IP/Bitcoin addresses.
// It will periodically rotate the files and supports compression.
type Recorder struct {
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

// New creates a new recorder with the given options.
func NewRecorder(options ...func(*Recorder)) (*Recorder, error) {
	rec := &Recorder{
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
func SetLog(log adaptor.Log) func(*Recorder) {
	return func(rec *Recorder) {
		rec.log = log
	}
}

// SetTypes sets the type of events to write to file.
func FilterTypes(cmds ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

func FilterIPs(ips ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, ip := range ips {
			rec.ipConfig[ip] = true
		}
	}
}

func FilterAddresses(addrs ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, addr := range addrs {
			rec.addrConfig[addr] = true
		}
	}
}

func AddWriter(w adaptor.Writer) func(*Recorder) {
	return func(rec *Recorder) {
		rec.writers = append(rec.writers, w)
	}
}

// Message will process a given message and log it if it's elligible.
func (rec *Recorder) Message(msg wire.Message, ra *net.TCPAddr,
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

	var record Record

	switch m := msg.(type) {
	case *wire.MsgAddr:
		record = NewAddressRecord(m, ra, la)

	case *wire.MsgAlert:
		record = NewAlertRecord(m, ra, la)

	case *wire.MsgBlock:
		if rec.blockIndex.Has(m.BlockSha()) {
			return
		}

		rec.blockIndex.Insert(m.BlockSha())
		record = NewBlockRecord(m, ra, la)

	case *wire.MsgHeaders:
		record = NewHeadersRecord(m, ra, la)

	case *wire.MsgInv:
		record = NewInventoryRecord(m, ra, la)

	case *wire.MsgPing:
		record = NewPingRecord(m, ra, la)

	case *wire.MsgPong:
		record = NewPongRecord(m, ra, la)

	case *wire.MsgReject:
		record = NewRejectRecord(m, ra, la)

	case *wire.MsgVersion:
		record = NewVersionRecord(m, ra, la)

	case *wire.MsgTx:
		if rec.txIndex.Has(m.TxSha()) {
			return
		}

		rec.txIndex.Insert(m.TxSha())
		tx := NewTransactionRecord(m, ra, la)
		ok := true

		if len(rec.addrConfig) > 0 {
			ok = false
		Outer:
			for _, out := range tx.details.outs {
				for _, addr := range out.addrs {
					if rec.addrConfig[addr.EncodeAddress()] {
						ok = true
						break Outer
					}
				}
			}
		}

		if !ok {
			return
		}

		record = NewTransactionRecord(m, ra, la)

	case *wire.MsgFilterAdd:
		record = NewFilterAddRecord(m, ra, la)

	case *wire.MsgFilterClear:
		record = NewFilterClearRecord(m, ra, la)

	case *wire.MsgFilterLoad:
		record = NewFilterLoadRecord(m, ra, la)

	case *wire.MsgGetAddr:
		record = NewGetAddrRecord(m, ra, la)

	case *wire.MsgGetBlocks:
		record = NewGetBlocksRecord(m, ra, la)

	case *wire.MsgGetData:
		record = NewGetDataRecord(m, ra, la)

	case *wire.MsgGetHeaders:
		record = NewGetHeadersRecord(m, ra, la)

	case *wire.MsgMemPool:
		record = NewMemPoolRecord(m, ra, la)

	case *wire.MsgMerkleBlock:
		record = NewMerkleBlockRecord(m, ra, la)

	case *wire.MsgNotFound:
		record = NewNotFoundRecord(m, ra, la)

	case *wire.MsgVerAck:
		record = NewVerAckRecord(m, ra, la)
	}

	for _, writer := range rec.writers {
		writer.Line(record.String())
	}
}
