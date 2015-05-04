package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterAddRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
}

func NewFilterAddRecord(msg *wire.MsgFilterAdd, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterAddRecord {
	record := &FilterAddRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgFilterAdd,
	}

	return record
}
