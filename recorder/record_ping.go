package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type PingRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	nonce uint64
}

func NewPingRecord(msg *wire.MsgPing, ra *net.TCPAddr,
	la *net.TCPAddr) *PingRecord {
	record := &PingRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		nonce: msg.Nonce,
	}

	return record
}
