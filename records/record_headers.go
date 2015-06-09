package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeadersRecord struct {
	Record

	hdrs []*HeaderRecord
}

func NewHeadersRecord(msg *wire.MsgHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *HeadersRecord {
	record := &HeadersRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		hdrs: make([]*HeaderRecord, len(msg.Headers)),
	}

	for i, hdr := range msg.Headers {
		record.hdrs[i] = NewHeaderRecord(hdr)
	}

	return record
}

func (hr *HeadersRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(hr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(hr.hdrs)), 10))

	for _, hdr := range hr.hdrs {
		buf.WriteString(Delimiter2)
		buf.WriteString(hdr.String())
	}

	return buf.String()
}
