package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MemPoolRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewMemPoolRecord(msg *wire.MsgMemPool, ra *net.TCPAddr,
	la *net.TCPAddr) *MemPoolRecord {
	record := &MemPoolRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (mr *MemPoolRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(mr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(mr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(mr.la.String())
	buf.WriteString(" ")
	buf.WriteString(mr.cmd)

	return buf.String()
}

func (hr *MemPoolRecord) Bytes() []byte {
	return make([]byte, 0)
}
