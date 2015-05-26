package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MerkleBlockRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
}

func NewMerkleBlockRecord(msg *wire.MsgMerkleBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *MerkleBlockRecord {
	record := &MerkleBlockRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
	}

	return record
}

func (mr *MerkleBlockRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(mr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(mr.la.String())

	return buf.String()
}

func (mr *MerkleBlockRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(mr.cmd)) //  1
	binary.Write(buf, binary.LittleEndian, mr.stamp.UnixNano())  //  8
	binary.Write(buf, binary.LittleEndian, mr.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(mr.ra.Port))   //  2
	binary.Write(buf, binary.LittleEndian, mr.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(mr.la.Port))   //  2

	// total: 45
	return buf.Bytes()
}
