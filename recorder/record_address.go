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
	stamp     time.Time
	la        *net.TCPAddr
	ra        *net.TCPAddr
	addr_list []*net.TCPAddr
}

func NewAddressRecord(msg *wire.MsgAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *AddressRecord {
	addr_list := make([]*net.TCPAddr, len(msg.AddrList))
	for i, na := range msg.AddrList {
		addr, err := util.ParseNetAddress(na)
		if err != nil {
			addr = &net.TCPAddr{IP: net.IPv4zero, Port: 0}
		}

		addr_list[i] = addr
	}

	ar := &AddressRecord{
		stamp:     time.Now(),
		ra:        ra,
		la:        la,
		addr_list: addr_list,
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
	buf.WriteString(" addr ")
	buf.WriteString(strconv.FormatInt(int64(len(ar.addr_list)), 10))

	for _, addr := range ar.addr_list {
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
	binary.Write(buf, binary.LittleEndian, len(ar.addr_list))

	for _, addr := range ar.addr_list {
		binary.Write(buf, binary.LittleEndian, addr.IP)
		binary.Write(buf, binary.LittleEndian, addr.Port)
	}

	return buf.Bytes()
}
