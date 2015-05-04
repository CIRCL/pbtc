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
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
	hash  [32]byte
	ins   []*InputRecord
	outs  []*OutputRecord
}

func NewTransactionRecord(msg *wire.MsgTx, ra *net.TCPAddr,
	la *net.TCPAddr) *TransactionRecord {
	tr := &TransactionRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		hash:  [32]byte(msg.TxSha()),
		ins:   make([]*InputRecord, len(msg.TxIn)),
		outs:  make([]*OutputRecord, len(msg.TxOut)),
	}

	for i, txin := range msg.TxIn {
		tr.ins[i] = NewInputRecord(txin)
	}

	for i, txout := range msg.TxOut {
		tr.outs[i] = NewOutputRecord(txout)
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
	buf.WriteString(strconv.Itoa(len(tr.ins)))
	buf.WriteString(" ")
	buf.WriteString(strconv.Itoa(len(tr.outs)))

	for _, input := range tr.ins {
		buf.WriteString("\n")
		buf.WriteString(input.String())
	}

	for _, output := range tr.outs {
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
	binary.Write(buf, binary.LittleEndian, len(tr.ins))
	binary.Write(buf, binary.LittleEndian, len(tr.outs))

	for _, input := range tr.ins {
		binary.Write(buf, binary.LittleEndian, input.Bytes())
	}

	for _, output := range tr.outs {
		binary.Write(buf, binary.LittleEndian, output.Bytes())
	}

	return buf.Bytes()
}
