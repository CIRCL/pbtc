package records

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MemPoolRecord struct {
	Record
}

func NewMemPoolRecord(msg *wire.MsgMemPool, ra *net.TCPAddr,
	la *net.TCPAddr) *MemPoolRecord {
	record := &MemPoolRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},
	}

	return record
}

func (mr *MemPoolRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(mr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.la.String())

	return buf.String()
}
