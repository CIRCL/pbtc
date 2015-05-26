package records

import (
	"bytes"
	"encoding/binary"
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
	buf.WriteString(vr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.la.String())

	return buf.String()
}

func (vr *VerAckRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(vr.cmd)) //  1
	binary.Write(buf, binary.LittleEndian, vr.stamp.UnixNano())  //  8
	binary.Write(buf, binary.LittleEndian, vr.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.ra.Port))   //  2
	binary.Write(buf, binary.LittleEndian, vr.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.la.Port))   //  2

	// total: 45
	return buf.Bytes()
}
