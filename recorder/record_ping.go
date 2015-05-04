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
	msg_t MsgType
	nonce uint64
}

func NewPingRecord(msg *wire.MsgPing, ra *net.TCPAddr,
	la *net.TCPAddr) *PingRecord {
	record := &PingRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgPing,
		nonce: msg.Nonce,
	}

	return record
}
