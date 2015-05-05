package recorder

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetAddrRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewGetAddrRecord(msg *wire.MsgGetAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *GetAddrRecord {
	record := &GetAddrRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (gr *GetAddrRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(gr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())
	buf.WriteString(" ")
	buf.WriteString(gr.cmd)

	return buf.String()
}
