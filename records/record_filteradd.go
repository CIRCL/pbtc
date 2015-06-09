package records

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterAddRecord struct {
	Record
}

func NewFilterAddRecord(msg *wire.MsgFilterAdd, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterAddRecord {
	record := &FilterAddRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},
	}

	return record
}

func (fr *FilterAddRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(fr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(fr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(fr.la.String())

	return buf.String()
}
