package records

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type ItemRecord struct {
	category uint8
	hash     [32]byte
}

func NewItemRecord(vec *wire.InvVect) *ItemRecord {
	ir := &ItemRecord{
		category: uint8(vec.Type),
		hash:     vec.Hash,
	}

	return ir
}

func (ir *ItemRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatUint(uint64(ir.category), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(hex.EncodeToString(ir.hash[:]))

	return buf.String()
}
