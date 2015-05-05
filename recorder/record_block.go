package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	hdr   *HeaderRecord
	txs   []*TransactionRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		hdr:   NewHeaderRecord(&msg.Header),
		txs:   make([]*TransactionRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.txs[i] = NewTransactionRecord(tx, ra, la)
	}

	return record
}

func (br *BlockRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(br.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(br.ra.String())
	buf.WriteString(" ")
	buf.WriteString(br.la.String())
	buf.WriteString(" ")
	buf.WriteString(br.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(br.txs)), 10))
	buf.WriteString("\n")
	buf.WriteString(br.hdr.String())

	for _, tr := range br.txs {
		buf.WriteString("\n")
		buf.WriteString(tr.String())
	}

	return buf.String()
}
