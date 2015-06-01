package records

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

func (gr *GetAddrRecord) Address() *net.TCPAddr {
	return gr.ra
}

func (gr *GetAddrRecord) Cmd() string {
	return gr.cmd
}

func (gr *GetAddrRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.la.String())

	return buf.String()
}