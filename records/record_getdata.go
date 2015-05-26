package records

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

	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(gr.items)), 10))

	for _, item := range gr.items {
		buf.WriteString(Delimiter2)
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (gr *GetDataRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(gr.cmd))  //  1
	binary.Write(buf, binary.LittleEndian, gr.stamp.UnixNano())   //  8
	binary.Write(buf, binary.LittleEndian, gr.ra.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(gr.ra.Port))    //  2
	binary.Write(buf, binary.LittleEndian, gr.la.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(gr.la.Port))    //  2
	binary.Write(buf, binary.LittleEndian, uint16(len(gr.items))) //  2

	for _, item := range gr.items { // N
		binary.Write(buf, binary.LittleEndian, item.Bytes()) // 33
	}

	// total: 47 + N*33
	return buf.Bytes()
}
