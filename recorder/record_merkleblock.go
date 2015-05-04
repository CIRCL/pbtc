package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MerkleBlockRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
}

func NewMerkleBlockRecord(msg *wire.MsgMerkleBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *MerkleBlockRecord {
	record := &MerkleBlockRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgMerkleBlock,
	}

	return record
}
