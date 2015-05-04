package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterLoadRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
}

func NewFilterLoadRecord(msg *wire.MsgFilterLoad, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterLoadRecord {
	record := &FilterLoadRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgFilterLoad,
	}

	return record
}
