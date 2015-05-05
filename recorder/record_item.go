package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type ItemRecord struct {
	category uint32
	hash     []byte
}

func NewItemRecord(vec *wire.InvVect) *ItemRecord {
	ir := &ItemRecord{
		category: uint32(vec.Type),
		hash:     vec.Hash.Bytes(),
	}

	return ir
}

func (ir *ItemRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatUint(uint64(ir.category), 10))
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(ir.hash))

	return buf.String()
}

func (ir *ItemRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ir.category)
	binary.Write(buf, binary.LittleEndian, ir.hash)

	return buf.Bytes()
}
