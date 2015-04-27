package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/util"
)

type VersionRecord struct {
	stamp    time.Time
	version  int32
	services uint64
	rstamp   time.Time
	remote   *net.TCPAddr
	local    *net.TCPAddr
	nonce    uint64
	agent    string
	block    int32
	relay    bool
}

func NewVersionRecord(msg *wire.MsgVersion) *VersionRecord {
	vr := &VersionRecord{
		stamp:    time.Now(),
		version:  msg.ProtocolVersion,
		services: uint64(msg.Services),
		rstamp:   msg.Timestamp,
		remote:   util.ParseNetAddress(&msg.AddrYou),
		local:    util.ParseNetAddress(&msg.AddrMe),
		nonce:    msg.Nonce,
		agent:    msg.UserAgent,
		block:    msg.LastBlock,
		relay:    !msg.DisableRelayTx,
	}

	return vr
}

func (vr *VersionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("ver ")
	buf.WriteString(vr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(vr.version), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(uint64(vr.services), 10))
	buf.WriteString(" ")
	buf.WriteString(vr.rstamp.String())
	buf.WriteString(" ")
	buf.WriteString(vr.remote.String())
	buf.WriteString(" ")
	buf.WriteString(vr.local.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(vr.nonce, 10))
	buf.WriteString(" ")
	buf.WriteString(vr.agent)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(vr.block), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatBool(vr.relay))

	return buf.String()
}

func (vr *VersionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, wire.CmdVersion)
	binary.Write(buf, binary.LittleEndian, vr.stamp)
	binary.Write(buf, binary.LittleEndian, vr.version)
	binary.Write(buf, binary.LittleEndian, vr.services)
	binary.Write(buf, binary.LittleEndian, vr.rstamp)
	binary.Write(buf, binary.LittleEndian, vr.remote.IP.To16())
	binary.Write(buf, binary.LittleEndian, int16(vr.remote.Port))
	binary.Write(buf, binary.LittleEndian, vr.local.IP.To16())
	binary.Write(buf, binary.LittleEndian, int16(vr.local.Port))
	binary.Write(buf, binary.LittleEndian, vr.nonce)
	binary.Write(buf, binary.LittleEndian, vr.agent)
	binary.Write(buf, binary.LittleEndian, vr.block)
	binary.Write(buf, binary.LittleEndian, vr.relay)

	return buf.Bytes()
}
