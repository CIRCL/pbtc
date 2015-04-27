package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type TransactionRecord struct {
	hash     [32]byte
	in_list  []*OutputRecord
	out_list []*OutputRecord
}

func NewTransactionRecord(msg *wire.MsgTx) *TransactionRecord {
	in_list := make([]*OutputRecord, len(msg.TxIn))
	for i, txin := range msg.TxIn {
		_ = txin
		in_list[i], _ = NewOutputRecord(nil)
	}

	out_list := make([]*OutputRecord, len(msg.TxOut))
	for i, txout := range msg.TxOut {
		out_list[i], _ = NewOutputRecord(txout)
	}

	tr := &TransactionRecord{
		hash:     [32]byte(msg.TxSha()),
		in_list:  in_list,
		out_list: out_list,
	}

	return tr
}

func (tr *TransactionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("tx ")
	buf.WriteString(hex.Dump(tr.hash[:]))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.in_list)))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.out_list)))
	buf.WriteString("\n")
	for _, partial := range tr.in_list {
		buf.WriteString(partial.String())
		buf.WriteString("\n")
	}
	for _, partial := range tr.out_list {
		buf.WriteString(partial.String())
		buf.WriteString("\n")
	}

	return buf.String()
}

func (tr *TransactionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, wire.CmdTx)
	binary.Write(buf, binary.LittleEndian, tr.hash)
	binary.Write(buf, binary.LittleEndian, len(tr.in_list))
	binary.Write(buf, binary.LittleEndian, len(tr.out_list))
	for _, partial := range tr.in_list {
		binary.Write(buf, binary.LittleEndian, partial.Bytes())
	}
	for _, partial := range tr.out_list {
		binary.Write(buf, binary.LittleEndian, partial.Bytes())
	}

	return buf.Bytes()
}
