package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type InventoryRecord struct{}

func NewInventoryRecord(msg *wire.MsgInv) *InventoryRecord {
	return &InventoryRecord{}
}

func (ir *InventoryRecord) String() string {
	return ""
}
