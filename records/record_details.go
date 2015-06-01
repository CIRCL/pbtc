package records

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type DetailsRecord struct {
	hash [32]byte
	ins  []*InputRecord
	outs []*OutputRecord
}

func NewDetailsRecord(msg *wire.MsgTx) *DetailsRecord {
	record := &DetailsRecord{
		hash: msg.TxSha(),
		ins:  make([]*InputRecord, len(msg.TxIn)),
		outs: make([]*OutputRecord, len(msg.TxOut)),
	}

	for i, txin := range msg.TxIn {
		record.ins[i] = NewInputRecord(txin)
	}

	for i, txout := range msg.TxOut {
		record.outs[i] = NewOutputRecord(txout)
	}

	return record
}

func (dr *DetailsRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(hex.EncodeToString(dr.hash[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(len(dr.ins)), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(len(dr.outs)), 10))

	for _, input := range dr.ins {
		buf.WriteString(Delimiter2)
		buf.WriteString(input.String())
	}

	for _, output := range dr.outs {
		buf.WriteString(Delimiter2)
		buf.WriteString(output.String())
	}

	return buf.String()
}
