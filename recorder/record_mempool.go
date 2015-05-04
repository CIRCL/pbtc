package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type MemPoolRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
}

func NewMemPoolRecord(msg *wire.MsgMemPool, ra *net.TCPAddr,
	la *net.TCPAddr) *MemPoolRecord {
	record := &MemPoolRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgMemPool,
	}

	return record
}
