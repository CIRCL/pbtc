package records

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type TransactionRecord struct {
	Record

	details *DetailsRecord
}

func NewTransactionRecord(msg *wire.MsgTx, ra *net.TCPAddr,
	la *net.TCPAddr) *TransactionRecord {
	record := &TransactionRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		details: NewDetailsRecord(msg),
	}

	return record
}

func (tr *TransactionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(tr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.details.String())

	return buf.String()
}

func (tr *TransactionRecord) HasAddress(addr string) bool {
	for _, out := range tr.details.outs {
		for _, a := range out.addrs {
			if a.EncodeAddress() == addr {
				return true
			}
		}
	}

	return false
}
