package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MemPoolRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewMemPoolRecord(msg *wire.MsgMemPool, ra *net.TCPAddr,
	la *net.TCPAddr) *MemPoolRecord {
	record := &MemPoolRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (mr *MemPoolRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(mr.cmd)
	buf.WriteString(" ")
	buf.WriteString(mr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(mr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(mr.la.String())

	return buf.String()
}

func (mr *MemPoolRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, mr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, mr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(mr.ra.Port))
	binary.Write(buf, binary.LittleEndian, mr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(mr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(mr.cmd))

	return buf.Bytes()
}
