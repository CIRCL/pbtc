package records

import (
	"bytes"
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterClearRecord struct {
	Record
}

func NewFilterClearRecord(msg *wire.MsgFilterClear, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterClearRecord {
	record := &FilterClearRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},
	}

	return record
}

func (fr *FilterClearRecord) String() string {
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
