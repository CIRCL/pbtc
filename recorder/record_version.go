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
	ra       *net.TCPAddr
	la       *net.TCPAddr
	stamp    time.Time
	msg_t    MsgType
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

func NewVersionRecord(msg *wire.MsgVersion, ra *net.TCPAddr,
	la *net.TCPAddr) *VersionRecord {

	remote, err := util.ParseNetAddress(&msg.AddrYou)
	if err != nil {
		remote = &net.TCPAddr{IP: net.IPv4zero, Port: 0}
	}

	local, err := util.ParseNetAddress(&msg.AddrMe)
	if err != nil {
		local = &net.TCPAddr{IP: net.IPv4zero, Port: 0}
	}

	vr := &VersionRecord{
		stamp:    time.Now(),
		ra:       ra,
		la:       la,
		msg_t:    MsgVersion,
		version:  msg.ProtocolVersion,
		services: uint64(msg.Services),
		rstamp:   msg.Timestamp,
		remote:   remote,
		local:    local,
		nonce:    msg.Nonce,
		agent:    msg.UserAgent,
		block:    msg.LastBlock,
		relay:    !msg.DisableRelayTx,
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
	binary.Write(buf, binary.LittleEndian, vr.stamp.Unix())
	binary.Write(buf, binary.LittleEndian, vr.la.IP)
	binary.Write(buf, binary.LittleEndian, vr.la.Port)
	binary.Write(buf, binary.LittleEndian, vr.ra.IP)
	binary.Write(buf, binary.LittleEndian, vr.ra.Port)
	binary.Write(buf, binary.LittleEndian, wire.CmdVersion)
	binary.Write(buf, binary.LittleEndian, vr.version)
	binary.Write(buf, binary.LittleEndian, vr.services)
	binary.Write(buf, binary.LittleEndian, vr.rstamp.Unix())
	binary.Write(buf, binary.LittleEndian, vr.remote.IP)
	binary.Write(buf, binary.LittleEndian, vr.remote.Port)
	binary.Write(buf, binary.LittleEndian, vr.local.IP)
	binary.Write(buf, binary.LittleEndian, vr.local.Port)
	binary.Write(buf, binary.LittleEndian, vr.nonce)
	binary.Write(buf, binary.LittleEndian, vr.agent)
	binary.Write(buf, binary.LittleEndian, vr.block)
	binary.Write(buf, binary.LittleEndian, vr.relay)

	return buf.Bytes()
}
