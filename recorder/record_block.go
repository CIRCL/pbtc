package recorders

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	hdr   *HeaderRecord
	txs   []*TransactionRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		hdr:   NewHeaderRecord(msg.Header),
		txs:   make([]*TransactionRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.txs[i] = NewTransactionRecord(tx, ra, la)
	}

	return record
}
