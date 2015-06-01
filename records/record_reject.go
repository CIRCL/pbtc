package records

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

func (rr *RejectRecord) Address() *net.TCPAddr {
	return rr.ra
}

func (rr *RejectRecord) Cmd() string {
	return rr.cmd
}

func (rr *RejectRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(rr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(rr.code), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reject)
	buf.WriteString(Delimiter1)
	buf.WriteString(hex.EncodeToString(rr.hash))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reason)

	return buf.String()
}
