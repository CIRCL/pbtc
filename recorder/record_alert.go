package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type AlertRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	string text
	id     uint32
	cancel uint32
	expire uint64
	minver int32
	maxver int32
}

func NewAlertRecord(msg *wire.MsgAlert, ra *net.TCPAddr,
	la *net.TCPAddr) *AlertRecord {
	record := &AlertRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		id:     msg.Payload.ID,
		text:   msg.Payload.Comment,
		expire: msg.Payload.Expiration,
		cancel: msg.Payload.Cancel,
		minver: msg.Payload.MinVer,
		maxver: msg.Payload.MaxVer,
	}

	return record
}
