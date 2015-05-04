package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterClearRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
}

func NewFilterClearRecord(msg *wire.MsgFilterClear, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterClearRecord {
	record := &FilterClearRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgFilterClear,
	}

	return record
}
