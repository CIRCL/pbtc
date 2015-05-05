package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetDataRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	items []*ItemRecord
}

func NewGetDataRecord(msg *wire.MsgGetData, ra *net.TCPAddr,
	la *net.TCPAddr) *GetDataRecord {
	record := &GetDataRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		items: make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.items[i] = NewItemRecord(item)
	}

	return record
}

func (gr *GetDataRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(gr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())
	buf.WriteString(" ")
	buf.WriteString(gr.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(gr.items)), 10))

	for _, item := range gr.items {
		buf.WriteString("\n")
		buf.WriteString(item.String())
	}

	return buf.String()
}
