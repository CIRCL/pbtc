package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetDataRecord struct {
	Record

	items []*ItemRecord
}

func NewGetDataRecord(msg *wire.MsgGetData, ra *net.TCPAddr,
	la *net.TCPAddr) *GetDataRecord {
	record := &GetDataRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		items: make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.items[i] = NewItemRecord(item)
	}

	return record
}

func (gr *GetDataRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(gr.items)), 10))

	for _, item := range gr.items {
		buf.WriteString(Delimiter2)
		buf.WriteString(item.String())
	}

	return buf.String()
}
