package recorder

import (
	"bytes"
	"encoding/binary"
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

func (hr *GetAddrRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, hr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, hr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(hr.ra.Port))
	binary.Write(buf, binary.LittleEndian, hr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(hr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(hr.cmd))

	return buf.Bytes()
}
