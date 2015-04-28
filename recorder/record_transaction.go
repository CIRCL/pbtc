package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type TransactionRecord struct {
	stamp    time.Time
	ra       *net.TCPAddr
	la       *net.TCPAddr
	hash     [32]byte
	in_list  []*InputRecord
	out_list []*OutputRecord
}

func NewTransactionRecord(msg *wire.MsgTx, ra *net.TCPAddr,
	la *net.TCPAddr) *TransactionRecord {
	in_list := make([]*InputRecord, len(msg.TxIn))
	for i, txin := range msg.TxIn {
		in_list[i] = NewInputRecord(txin)
	}

	out_list := make([]*OutputRecord, len(msg.TxOut))
	for i, txout := range msg.TxOut {
		out_list[i] = NewOutputRecord(txout)
	}

	tr := &TransactionRecord{
		stamp:    time.Now(),
		ra:       ra,
		la:       la,
		hash:     [32]byte(msg.TxSha()),
		in_list:  in_list,
		out_list: out_list,
	}

	return tr
}

func (tr *TransactionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(tr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(tr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(tr.la.String())
	buf.WriteString(" tx ")
	buf.WriteString(hex.EncodeToString(tr.hash[:]))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.in_list)))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.out_list)))

	for _, input := range tr.in_list {
		buf.WriteString("\n")
		buf.WriteString(input.String())
	}

	for _, output := range tr.out_list {
		buf.WriteString("\n")
		buf.WriteString(output.String())
	}

	return buf.String()
}

func (tr *TransactionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tr.stamp.Unix())
	binary.Write(buf, binary.LittleEndian, tr.ra.IP)
	binary.Write(buf, binary.LittleEndian, tr.ra.Port)
	binary.Write(buf, binary.LittleEndian, tr.la.IP)
	binary.Write(buf, binary.LittleEndian, tr.la.Port)
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
