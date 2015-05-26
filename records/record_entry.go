package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/util"
)

type EntryRecord struct {
	addr     *net.TCPAddr
	stamp    time.Time
	services uint64
}

func NewEntryRecord(na *wire.NetAddress) *EntryRecord {
	record := &EntryRecord{
		addr:     util.ParseNetAddress(na),
		stamp:    na.Timestamp,
		services: uint64(na.Services),
	}

	return record
}

func (ar *EntryRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(strconv.FormatInt(ar.stamp.Unix(), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(ar.services, 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(ar.addr.String())

	return buf.String()
}

func (ar *EntryRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(ar.stamp.Unix())) //  4
	binary.Write(buf, binary.LittleEndian, ar.services)             //  8
	binary.Write(buf, binary.LittleEndian, ar.addr.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(ar.addr.Port))    //  2

	// total: 30 bytes
	return buf.Bytes()
}
