package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type PongRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	nonce uint64
}

func NewPongRecord(msg *wire.MsgPong, ra *net.TCPAddr,
	la *net.TCPAddr) *PongRecord {
	record := &PongRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		nonce: msg.Nonce,
	}

	return record
}
