package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetAddrRecord struct{}

func NewGetAddrRecord(msg *wire.MsgGetAddr, ra *net.TCPAddr,
	la *net.TCPAddr) *GetAddrRecord {
	record := &GetAddrRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
	}

	return record
}
