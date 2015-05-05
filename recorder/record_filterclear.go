package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterClearRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewFilterClearRecord(msg *wire.MsgFilterClear, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterClearRecord {
	record := &FilterClearRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (fr *FilterClearRecord) String() string {
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

func (hr *FilterClearRecord) Bytes() []byte {
	return make([]byte, 0)
}
