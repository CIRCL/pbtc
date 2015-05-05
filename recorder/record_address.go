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

type AddressRecord struct {
	stamp time.Time
	la    *net.TCPAddr
	ra    *net.TCPAddr
	cmd   string
	addrs []*net.TCPAddr
}

func NewAddressRecord(msg *wire.MsgAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *AddressRecord {
	ar := &AddressRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		addrs: make([]*net.TCPAddr, len(msg.AddrList)),
	}

	for i, na := range msg.AddrList {
		ar.addrs[i] = util.ParseNetAddress(na)
	}

	return ar
}

func (ar *AddressRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(ar.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(ar.ra.String())
	buf.WriteString(" ")
	buf.WriteString(ar.la.String())
	buf.WriteString(" ")
	buf.WriteString(ar.cmd)
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(ar.addrs)), 10))

	for _, addr := range ar.addrs {
		buf.WriteString(" ")
		buf.WriteString(addr.String())
	}

	return buf.String()
}

func (ar *AddressRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ar.stamp.Unix())
	binary.Write(buf, binary.LittleEndian, ar.ra.IP)
	binary.Write(buf, binary.LittleEndian, ar.ra.Port)
	binary.Write(buf, binary.LittleEndian, ar.la.IP)
	binary.Write(buf, binary.LittleEndian, ar.la.Port)
	binary.Write(buf, binary.LittleEndian, wire.CmdAddr)
	binary.Write(buf, binary.LittleEndian, len(ar.addrs))

	for _, addr := range ar.addrs {
		binary.Write(buf, binary.LittleEndian, addr.IP)
		binary.Write(buf, binary.LittleEndian, addr.Port)
	}

	return buf.Bytes()
}
