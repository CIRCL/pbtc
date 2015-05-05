package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type PingRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	nonce uint64
}

func NewPingRecord(msg *wire.MsgPing, ra *net.TCPAddr,
	la *net.TCPAddr) *PingRecord {
	record := &PingRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		nonce: msg.Nonce,
	}

	return record
}

func (pr *PingRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(pr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(pr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(pr.la.String())
	buf.WriteString(" ")
	buf.WriteString(pr.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(pr.nonce, 10))

	return buf.String()
}

func (hr *PingRecord) Bytes() []byte {
	return make([]byte, 0)
}
