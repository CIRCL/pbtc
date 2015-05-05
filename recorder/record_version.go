package recorder

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/util"
)

type VersionRecord struct {
	ra       *net.TCPAddr
	la       *net.TCPAddr
	stamp    time.Time
	cmd      string
	version  int32
	services uint64
	sent     time.Time
	raddr    *net.TCPAddr
	laddr    *net.TCPAddr
	agent    string
	block    int32
	relay    bool
	nonce    uint64
}

func NewVersionRecord(msg *wire.MsgVersion, ra *net.TCPAddr,
	la *net.TCPAddr) *VersionRecord {
	vr := &VersionRecord{
		stamp:    time.Now(),
		ra:       ra,
		la:       la,
		cmd:      msg.Command(),
		version:  msg.ProtocolVersion,
		services: uint64(msg.Services),
		sent:     msg.Timestamp,
		raddr:    util.ParseNetAddress(&msg.AddrYou),
		laddr:    util.ParseNetAddress(&msg.AddrMe),
		agent:    msg.UserAgent,
		block:    msg.LastBlock,
		relay:    !msg.DisableRelayTx,
		nonce:    msg.Nonce,
	}

	return vr
}

func (vr *VersionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(vr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(vr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(vr.la.String())
	buf.WriteString(" ver ")
	buf.WriteString(strconv.FormatInt(int64(vr.version), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(vr.services, 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(vr.sent.Unix(), 10))
	buf.WriteString(" ")
	buf.WriteString(vr.raddr.String())
	buf.WriteString(" ")
	buf.WriteString(vr.laddr.String())
	buf.WriteString(" ")
	buf.WriteString(vr.agent)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(vr.block), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatBool(vr.relay))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(vr.nonce, 10))

	return buf.String()
}

func (vr *VersionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)

	return buf.Bytes()
}
