package records

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

func (pr *PingRecord) Address() *net.TCPAddr {
	return pr.ra
}

func (pr *PingRecord) Cmd() string {
	return pr.cmd
}

func (pr *PingRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(pr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(pr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(pr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(pr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatUint(pr.nonce, 10))

	return buf.String()
}