package recorder

import (
	"bytes"
	"encoding/binary"
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

func (ir *InventoryRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(ir.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(ir.ra.String())
	buf.WriteString(" ")
	buf.WriteString(ir.la.String())
	buf.WriteString(" ")
	buf.WriteString(ir.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(ir.inv)), 10))

	for _, item := range ir.inv {
		buf.WriteString("\n")
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (ir *InventoryRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ir.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, ir.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(ir.ra.Port))
	binary.Write(buf, binary.LittleEndian, ir.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(ir.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(ir.cmd))
	binary.Write(buf, binary.LittleEndian, len(ir.inv))

	for _, item := range ir.inv {
		binary.Write(buf, binary.LittleEndian, item.Bytes())
	}

	return buf.Bytes()
}
