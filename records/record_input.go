package records

import (
	"bytes"
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
