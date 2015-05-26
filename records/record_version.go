package records

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
	buf.WriteString(vr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(vr.version), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatUint(vr.services, 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(vr.sent.Unix(), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.raddr.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.laddr.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(vr.block), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatBool(vr.relay))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatUint(vr.nonce, 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(vr.agent)

	return buf.String()
}

func (vr *VersionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(vr.cmd))  //  1
	binary.Write(buf, binary.LittleEndian, vr.stamp.UnixNano())   //  8
	binary.Write(buf, binary.LittleEndian, vr.ra.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.ra.Port))    //  2
	binary.Write(buf, binary.LittleEndian, vr.la.IP.To16())       // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.la.Port))    //  2
	binary.Write(buf, binary.LittleEndian, vr.version)            //  4
	binary.Write(buf, binary.LittleEndian, vr.services)           //  8
	binary.Write(buf, binary.LittleEndian, vr.sent.Unix())        //  8
	binary.Write(buf, binary.LittleEndian, vr.raddr.IP.To16())    // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.raddr.Port)) //  2
	binary.Write(buf, binary.LittleEndian, vr.laddr.IP.To16())    // 16
	binary.Write(buf, binary.LittleEndian, uint16(vr.laddr.Port)) //  2
	binary.Write(buf, binary.LittleEndian, vr.block)              //  4
	binary.Write(buf, binary.LittleEndian, vr.relay)              //  1
	binary.Write(buf, binary.LittleEndian, vr.nonce)              //  8
	binary.Write(buf, binary.LittleEndian, uint32(len(vr.agent))) //  4
	binary.Write(buf, binary.LittleEndian, vr.agent)              //  X

	// total: 114 + X
	return buf.Bytes()
}
