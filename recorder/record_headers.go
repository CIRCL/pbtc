package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeadersRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	hdrs  []*HeaderRecord
}

func NewHeadersRecord(msg *wire.MsgHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *HeadersRecord {
	record := &HeadersRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		hdrs:  make([]*HeaderRecord, len(msg.Headers)),
	}

	for i, hdr := range msg.Headers {
		record.hdrs[i] = NewHeaderRecord(hdr)
	}

	return record
}

func (hr *HeadersRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(hr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(hr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(hr.la.String())
	buf.WriteString(" ")
	buf.WriteString(hr.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(hr.hdrs)), 10))

	for _, hdr := range hr.hdrs {
		buf.WriteString("\n")
		buf.WriteString(hdr.String())
	}

	return buf.String()
}

func (hr *HeadersRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, hr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, hr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(hr.ra.Port))
	binary.Write(buf, binary.LittleEndian, hr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(hr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(hr.cmd))
	binary.Write(buf, binary.LittleEndian, len(hr.hdrs))

	for _, hdr := range hr.hdrs {
		binary.Write(buf, binary.LittleEndian, hdr.Bytes())
	}

	return buf.Bytes()
}
