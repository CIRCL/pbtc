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

func (ir *InventoryRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(ir.cmd)) // 1
	binary.Write(buf, binary.LittleEndian, ir.stamp.UnixNano())  // 8
	binary.Write(buf, binary.LittleEndian, ir.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(ir.ra.Port))   // 2
	binary.Write(buf, binary.LittleEndian, ir.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(ir.la.Port))   // 2
	binary.Write(buf, binary.LittleEndian, uint16(len(ir.inv)))  // 2

	for _, item := range ir.inv { // N
		binary.Write(buf, binary.LittleEndian, item.Bytes()) // 33
	}

	// total: 47 + N*33
	return buf.Bytes()
}
