package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type InputRecord struct {
	hash     [32]byte
	index    uint32
	sequence uint32
}

func NewInputRecord(txin *wire.TxIn) *InputRecord {
	ir := &InputRecord{
		hash:     txin.PreviousOutPoint.Hash,
		index:    txin.PreviousOutPoint.Index,
		sequence: txin.Sequence,
	}

	return ir
}

func (ir *InputRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(hex.EncodeToString(ir.hash[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(ir.index), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(ir.sequence), 10))

	return buf.String()
}

func (ir *InputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ir.hash)     // 32
	binary.Write(buf, binary.LittleEndian, ir.index)    //  4
	binary.Write(buf, binary.LittleEndian, ir.sequence) //  4

	// total: 40
	return buf.Bytes()
}
