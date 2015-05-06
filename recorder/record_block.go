package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	hdr   *HeaderRecord
	txs   []*TransactionRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		hdr:   NewHeaderRecord(&msg.Header),
		txs:   make([]*TransactionRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.txs[i] = NewTransactionRecord(tx, ra, la)
	}

	return record
}

func (br *BlockRecord) String() string {
	buf := new(bytes.Buffer)

	// line 1: header + block header information + tx number
	buf.WriteString(br.cmd)
	buf.WriteString(" ")
	buf.WriteString(br.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(br.ra.String())
	buf.WriteString(" ")
	buf.WriteString(br.la.String())
	buf.WriteString(" ")
	buf.WriteString(br.hdr.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(br.txs)), 10))

	// should we add transaction summaries here ??

	return buf.String()
}

func (br *BlockRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, br.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, br.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(br.ra.Port))
	binary.Write(buf, binary.LittleEndian, br.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(br.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(br.cmd))
	binary.Write(buf, binary.LittleEndian, br.hdr.Bytes())
	binary.Write(buf, binary.LittleEndian, len(br.txs))

	for _, tx := range br.txs {
		binary.Write(buf, binary.LittleEndian, tx.Bytes())
	}

	return buf.Bytes()
}
