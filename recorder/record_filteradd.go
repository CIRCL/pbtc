package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterAddRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewFilterAddRecord(msg *wire.MsgFilterAdd, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterAddRecord {
	record := &FilterAddRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (fr *FilterAddRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(fr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(fr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(fr.la.String())
	buf.WriteString(" ")
	buf.WriteString(fr.cmd)

	return buf.String()
}

func (hr *FilterAddRecord) Bytes() []byte {
	return make([]byte, 0)
}
