package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type InventoryRecord struct{}

func NewInventoryRecord(inv *wire.InvVect) *InventoryRecord {
	return &InventoryRecord{}
}

func (record *InventoryRecord) String() string {
	return ""
}
