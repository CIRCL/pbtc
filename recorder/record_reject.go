package recorder

import (
	"bytes"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type RejectRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	cmd    string
	code   uint8
	reject string
	hash   []byte
	reason string
}

func NewRejectRecord(msg *wire.MsgReject, ra *net.TCPAddr,
	la *net.TCPAddr) *RejectRecord {
	record := &RejectRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		cmd:    msg.Command(),
		code:   uint8(msg.Code),
		reject: msg.Cmd,
		hash:   msg.Hash.Bytes(),
		reason: msg.Reason,
	}

	return record
}

func (rr *RejectRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(rr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(rr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(rr.la.String())
	buf.WriteString(" ")
	buf.WriteString(rr.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(rr.code), 10))
	buf.WriteString(" ")
	buf.WriteString(rr.reject)
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(rr.hash))
	buf.WriteString(" ")
	buf.WriteString(rr.reason)

	return buf.String()
}

func (hr *RejectRecord) Bytes() []byte {
	return make([]byte, 0)
}
