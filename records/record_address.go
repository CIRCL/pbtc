package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type AddressRecord struct {
	Record

	addrs []*EntryRecord
}

func NewAddressRecord(msg *wire.MsgAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *AddressRecord {
	ar := &AddressRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		addrs: make([]*EntryRecord, len(msg.AddrList)),
	}

	for i, na := range msg.AddrList {
		ar.addrs[i] = NewEntryRecord(na)
	}

	return ar
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
