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
	in_list  []*InputRecord
	out_list []*OutputRecord
}

func NewTransactionRecord(msg *wire.MsgTx) *TransactionRecord {
	in_list := make([]*InputRecord, len(msg.TxIn))
	for i, txin := range msg.TxIn {
		in_list[i] = NewInputRecord(txin)
	}

	out_list := make([]*OutputRecord, len(msg.TxOut))
	for i, txout := range msg.TxOut {
		out_list[i] = NewOutputRecord(txout)
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
	buf.WriteString(hex.EncodeToString(tr.hash[:]))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.in_list)))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.out_list)))
	buf.WriteString("\n")

	for _, input := range tr.in_list {
		buf.WriteString(input.String())
		buf.WriteString("\n")
	}

	for _, output := range tr.out_list {
		buf.WriteString(output.String())
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

	for _, input := range tr.in_list {
		binary.Write(buf, binary.LittleEndian, input.Bytes())
	}

	for _, output := range tr.out_list {
		binary.Write(buf, binary.LittleEndian, output.Bytes())
	}

	return buf.Bytes()
}
