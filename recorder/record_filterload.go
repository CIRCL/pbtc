package recorder

import (
	"bytes"
	"encoding/binary"
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

	// line 1: header
	buf.WriteString(fr.cmd)
	buf.WriteString(" ")
	buf.WriteString(fr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(fr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(fr.la.String())

	return buf.String()
}

func (fr *FilterLoadRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, fr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, fr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(fr.ra.Port))
	binary.Write(buf, binary.LittleEndian, fr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(fr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(fr.cmd))

	return buf.Bytes()
}
