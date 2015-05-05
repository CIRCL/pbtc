package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type VerAckRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewVerAckRecord(msg *wire.MsgVerAck, ra *net.TCPAddr,
	la *net.TCPAddr) *VerAckRecord {
	record := &VerAckRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (vr *VerAckRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(vr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(vr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(vr.la.String())
	buf.WriteString(" ")
	buf.WriteString(vr.cmd)

	return buf.String()
}

func (hr *VerAckRecord) Bytes() []byte {
	return make([]byte, 0)
}
