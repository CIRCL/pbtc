package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type VerAckRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
}

func NewVerAckRecord(msg *wire.MsgVerAck, ra *net.TCPAddr,
	la *net.TCPAddr) *VerAckRecord {
	record := &VerAckRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
	}

	return record
}
