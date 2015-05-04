package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeadersRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	msg_t MsgType
	hdrs  []*HeaderRecord
}

func NewHeadersRecord(msg *wire.MsgHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *HeadersRecord {
	record := &HeadersRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		msg_t: MsgHeaders,
		hdrs:  make([]*HeaderRecord, len(msg.Headers)),
	}

	for i, hdr := range msg.Headers {
		hdrs[i] = NewHeaderRecord(hdr)
	}

	return record
}
