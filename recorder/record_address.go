package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type AddressRecord struct {
	stamp time.Time
	la    *net.TCPAddr
	ra    *net.TCPAddr
	cmd   string

	addrs []*EntryRecord
}

func NewAddressRecord(msg *wire.MsgAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *AddressRecord {
	ar := &AddressRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),

		addrs: make([]*EntryRecord, len(msg.AddrList)),
	}

	for i, na := range msg.AddrList {
		ar.addrs[i] = NewEntryRecord(na)
	}

	return ar
}

func (ar *AddressRecord) Addr() *net.TCPAddr {
	return ar.ra
}

func (ar *AddressRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(ar.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(ar.addrs)), 10))

	for _, addr := range ar.addrs {
		buf.WriteString(Delimiter2)
		buf.WriteString(addr.String())
	}

	return buf.String()
}

func (ar *AddressRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	// header
	binary.Write(buf, binary.LittleEndian, ParseCommand(ar.cmd))  //  1
	binary.Write(buf, binary.LittleEndian, ar.stamp.UnixNano())   //  8
	binary.Write(buf, binary.LittleEndian, ar.ra.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(ar.ra.Port))    //  2
	binary.Write(buf, binary.LittleEndian, ar.la.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(ar.la.Port))    //  2
	binary.Write(buf, binary.LittleEndian, uint16(len(ar.addrs))) //  2

	for _, addr := range ar.addrs {
		binary.Write(buf, binary.LittleEndian, addr.Bytes()) // 30
	}

	// total: 47 + N*30
	return buf.Bytes()
}
