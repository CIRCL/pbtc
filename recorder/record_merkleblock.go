package recorder

import (
	"bytes"
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
	buf.WriteString(mr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(mr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(mr.la.String())
	buf.WriteString(" ")
	buf.WriteString(mr.cmd)

	return buf.String()
}

func (hr *MerkleBlockRecord) Bytes() []byte {
	return make([]byte, 0)
}
