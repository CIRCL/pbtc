package recorder

import (
	"bytes"

	"github.com/btcsuite/btcd/wire"
)

type InputRecord struct{}

func NewInputRecord(txin *wire.TxIn) *InputRecord {
	ir := &InputRecord{}

	return ir
}

func (ir *InputRecord) String() string {
	buf := new(bytes.Buffer)

	return buf.String()
}

func (ir *InputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)

	return buf.Bytes()
}
