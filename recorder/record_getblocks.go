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
	msg_t MsgType
}

func NewGetBlocksRecord(msg *wire.MsgGetBlocks, ra *net.TCPAddr,
	la *net.TCPAddr) *GetBlocksRecord {
	record := &GetBlocksRecord{
		stamp: time.Time,
		ra:    *net.TCPAddr,
		la:    *net.TCPAdr,
		msg_t: MsgGetBlock,
	}

	return record
}
