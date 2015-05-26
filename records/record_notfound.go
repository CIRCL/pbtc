package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type NotFoundRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	inv   []*ItemRecord
}

func NewNotFoundRecord(msg *wire.MsgNotFound, ra *net.TCPAddr,
	la *net.TCPAddr) *NotFoundRecord {
	record := &NotFoundRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		inv:   make([]*ItemRecord, len(msg.InvList)),
	}

	for i, item := range msg.InvList {
		record.inv[i] = NewItemRecord(item)
	}

	return record
}

func (nr *NotFoundRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(nr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(nr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(nr.inv)), 10))

	for _, item := range nr.inv {
		buf.WriteString(Delimiter2)
		buf.WriteString(item.String())
	}

	return buf.String()
}

func (nr *NotFoundRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(nr.cmd)) //  1
	binary.Write(buf, binary.LittleEndian, nr.stamp.UnixNano())  //  8
	binary.Write(buf, binary.LittleEndian, nr.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(nr.ra.Port))   //  2
	binary.Write(buf, binary.LittleEndian, nr.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(nr.la.Port))   //  2
	binary.Write(buf, binary.LittleEndian, uint16(len(nr.inv)))  //  2

	for _, item := range nr.inv { // N
		binary.Write(buf, binary.LittleEndian, item.Bytes()) // 33
	}

	// total: 47 + N*33
	return buf.Bytes()
}
