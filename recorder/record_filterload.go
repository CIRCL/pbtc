package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterLoadRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewFilterLoadRecord(msg *wire.MsgFilterLoad, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterLoadRecord {
	record := &FilterLoadRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (fr *FilterLoadRecord) String() string {
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

func (hr *FilterLoadRecord) Bytes() []byte {
	return make([]byte, 0)
}
