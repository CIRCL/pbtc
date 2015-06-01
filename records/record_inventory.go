package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type InventoryRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	inv   []*ItemRecord
}

func NewInventoryRecord(msg *wire.MsgInv, ra *net.TCPAddr,
	la *net.TCPAddr) *InventoryRecord {
	ir := &InventoryRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		inv:   make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		ir.inv[i] = NewItemRecord(item)
	}

	return ir
}

func (ir *InventoryRecord) Address() *net.TCPAddr {
	return ir.ra
}

func (ir *InventoryRecord) Cmd() string {
	return ir.cmd
}

func (ir *InventoryRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(ir.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(ir.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(ir.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(ir.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(ir.inv)), 10))

	for _, item := range ir.inv {
		buf.WriteString(Delimiter2)
		buf.WriteString(item.String())
	}

	return buf.String()
}
