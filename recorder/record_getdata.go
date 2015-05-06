package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetDataRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	items []*ItemRecord
}

func NewGetDataRecord(msg *wire.MsgGetData, ra *net.TCPAddr,
	la *net.TCPAddr) *GetDataRecord {
	record := &GetDataRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		items: make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.items[i] = NewItemRecord(item)
	}

	return record
}

func (gr *GetDataRecord) String() string {
	buf := new(bytes.Buffer)

	// line 1: header + inventory length
	buf.WriteString(gr.cmd)
	buf.WriteString(" ")
	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(gr.items)), 10))

	// line 2 - (n+1): inventory items
	for _, item := range gr.items {
		buf.WriteString("\n")
		buf.WriteString(" ")
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (gr *GetDataRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, gr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, gr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(gr.ra.Port))
	binary.Write(buf, binary.LittleEndian, gr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(gr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(gr.cmd))
	binary.Write(buf, binary.LittleEndian, len(gr.items))

	for _, item := range gr.items {
		binary.Write(buf, binary.LittleEndian, item.Bytes())
	}

	return buf.Bytes()
}
