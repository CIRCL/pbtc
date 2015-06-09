package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type NotFoundRecord struct {
	Record

	inv []*ItemRecord
}

func NewNotFoundRecord(msg *wire.MsgNotFound, ra *net.TCPAddr,
	la *net.TCPAddr) *NotFoundRecord {
	record := &NotFoundRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		inv: make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.inv[i] = NewItemRecord(item)
	}

	return record
}

func (nr *NotFoundRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(nr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(nr.inv)), 10))

	for _, item := range nr.inv {
		buf.WriteString(Delimiter2)
		buf.WriteString(item.String())
	}

	return buf.String()
}
