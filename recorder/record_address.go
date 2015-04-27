package recorder

import (
	"bytes"
	"net"
	"strconv"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/util"
)

type AddressRecord struct {
	addr_list []*net.TCPAddr
}

func NewAddressRecord(msg *wire.MsgAddr) *AddressRecord {
	addr_list := make([]*net.TCPAddr, len(msg.AddrList))
	for i, addr := range msg.AddrList {
		addr_list[i] = util.ParseNetAddress(addr)
	}

	ar := &AddressRecord{
		addr_list: addr_list,
	}

	return ar
}

func (ar *AddressRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("addr ")
	buf.WriteString(strconv.FormatInt(int64(len(ar.addr_list)), 10))
	for _, addr := range ar.addr_list {
		buf.WriteString(" ")
		buf.WriteString(addr.String())
	}

	return buf.String()
}
