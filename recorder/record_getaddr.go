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

	// line 1: header
	buf.WriteString(gr.cmd)
	buf.WriteString(" ")
	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())

	return buf.String()
}

func (hr *GetAddrRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(hr.cmd)) //  1
	binary.Write(buf, binary.LittleEndian, hr.stamp.UnixNano())  //  8
	binary.Write(buf, binary.LittleEndian, hr.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(hr.ra.Port))   //  2
	binary.Write(buf, binary.LittleEndian, hr.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(hr.la.Port))   //  2

	return buf.Bytes()
}
