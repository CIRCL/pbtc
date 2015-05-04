package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetBlocksRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
}

func NewGetBlocksRecord(msg *wire.MsgGetBlocks, ra *net.TCPAddr,
	la *net.TCPAddr) *GetBlocksRecord {
	record := &GetBlocksRecord{
		stamp: time.Time,
		ra:    *net.TCPAddr,
		la:    *net.TCPAdr,
	}

	return record
}
