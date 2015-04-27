package recorder

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type InventoryRecord struct {
	inv_list []*ItemRecord
}

func NewInventoryRecord(msg *wire.MsgInv) *InventoryRecord {
	inv_list := make([]*ItemRecord, len(msg.InvList))
	for i, inv := range msg.InvList {
		inv_list[i] = NewItemRecord(inv)
	}

	ir := &InventoryRecord{
		inv_list: inv_list,
	}

	return ir
}

func (ir *InventoryRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("inv ")
	buf.WriteString(strconv.FormatInt(int64(len(ir.inv_list)), 10))

	for _, item := range ir.inv_list {
		buf.WriteString("\n")
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (ir *InventoryRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, wire.CmdInv)
	binary.Write(buf, binary.LittleEndian, len(ir.inv_list))

	for _, item := range ir.inv_list {
		binary.Write(buf, binary.LittleEndian, item.Bytes())
	}

	return buf.Bytes()
}
