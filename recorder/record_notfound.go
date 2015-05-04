package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type NotFoundRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
	inv   []*InventoryRecord
}

func NewNotFoundRecord(msg *wire.MsgNotFound, ra *net.TCPAddr,
	la *net.TCPAddr) *NotFoundRecord {
	record := &NetFoundRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgNotFound,
		inv:   make([]*InventoryRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.inv[i] = NewInventoryRecord(item)
	}

	return record
}
