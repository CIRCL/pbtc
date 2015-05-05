package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type AlertRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	cmd    string
	id     int32
	cancel int32
	expire int64
	minver int32
	maxver int32
	text   string
}

func NewAlertRecord(msg *wire.MsgAlert, ra *net.TCPAddr,
	la *net.TCPAddr) *AlertRecord {
	record := &AlertRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		cmd:    msg.Command(),
		id:     msg.Payload.ID,
		text:   msg.Payload.Comment,
		expire: msg.Payload.Expiration,
		cancel: msg.Payload.Cancel,
		minver: msg.Payload.MinVer,
		maxver: msg.Payload.MaxVer,
	}

	return record
}

func (ar *AlertRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(ar.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(ar.ra.String())
	buf.WriteString(" ")
	buf.WriteString(ar.la.String())
	buf.WriteString(" ")
	buf.WriteString(ar.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(ar.id), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(ar.cancel), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(ar.expire, 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(ar.minver), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(ar.maxver), 10))
	buf.WriteString(" ")
	buf.WriteString("\"")
	buf.WriteString(ar.text)
	buf.WriteString("\"")

	return buf.String()
}
