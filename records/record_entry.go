package records

import (
	"bytes"
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

func (er *EntryRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(strconv.FormatInt(er.stamp.Unix(), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(er.services, 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(er.addr.String())

	return buf.String()
}
