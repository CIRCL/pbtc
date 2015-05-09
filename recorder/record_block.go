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
	stamp   time.Time
	ra      *net.TCPAddr
	la      *net.TCPAddr
	cmd     string
	hdr     *HeaderRecord
	details []*DetailsRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		stamp:   time.Now(),
		ra:      ra,
		la:      la,
		cmd:     msg.Command(),
		hdr:     NewHeaderRecord(&msg.Header),
		details: make([]*DetailsRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.details[i] = NewDetailsRecord(tx)
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
	buf.WriteString(strconv.FormatInt(int64(len(br.details)), 10))

	// show many lines of details for each transaction
	for _, tx := range br.details {
		buf.WriteString("\n")
		buf.WriteString(" ")
		buf.WriteString(tx.String())
	}

	return buf.String()
}

func (br *BlockRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(br.cmd))    //   1
	binary.Write(buf, binary.LittleEndian, br.stamp.UnixNano())     //   8
	binary.Write(buf, binary.LittleEndian, br.ra.IP.To16())         //  16
	binary.Write(buf, binary.LittleEndian, uint16(br.ra.Port))      //   2
	binary.Write(buf, binary.LittleEndian, br.la.IP.To16())         //  16
	binary.Write(buf, binary.LittleEndian, uint16(br.la.Port))      //   2
	binary.Write(buf, binary.LittleEndian, br.hdr.Bytes())          // 113
	binary.Write(buf, binary.LittleEndian, uint32(len(br.details))) //   4

	for _, tx := range br.details { // N
		binary.Write(buf, binary.LittleEndian, tx.Bytes()) // X
	}

	// total: 162 + N*X
	return buf.Bytes()
}
