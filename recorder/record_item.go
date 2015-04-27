package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type ItemRecord struct {
}

func NewItemRecord(vec *wire.InvVect) *ItemRecord {
	ir := &ItemRecord{}

	return ir
}

func (ir *ItemRecord) String() string {
	return ""
}

func (ir *ItemRecord) Bytes() []byte {
	return nil
}
