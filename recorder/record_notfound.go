package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type NotFoundRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	inv   []*ItemRecord
}

func NewNotFoundRecord(msg *wire.MsgNotFound, ra *net.TCPAddr,
	la *net.TCPAddr) *NotFoundRecord {
	record := &NotFoundRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		inv:   make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.inv[i] = NewItemRecord(item)
	}

	return record
}

func (nr *NotFoundRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(nr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(nr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(nr.la.String())
	buf.WriteString(" ")
	buf.WriteString(nr.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(nr.inv)), 10))

	for _, item := range nr.inv {
		buf.WriteString("\n")
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (hr *NotFoundRecord) Bytes() []byte {
	return make([]byte, 0)
}
