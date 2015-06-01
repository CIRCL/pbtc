package records

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

func (mr *MerkleBlockRecord) Address() *net.TCPAddr {
	return mr.ra
}

func (mr *MerkleBlockRecord) Cmd() string {
	return mr.cmd
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
