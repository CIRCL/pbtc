package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetDataRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	items []*ItemRecord
}

func NewGetDataRecord(msg *wire.MsgGetData, ra *net.TCPAddr,
	la *net.TCPAddr) *GetDataRecord {
	record := &GetDataRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		items: make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.items[i] = NewItemRecord(item)
	}

	return record
}
