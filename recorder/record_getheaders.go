package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetHeadersRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	msg_t  MsgType
	stop   [32]byte
	hashes [][32]byte
}

func NewGetHeadersRecord(msg *wire.MsgGetHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *GetHeadersRecord {
	record := &GetHeadersRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		msg_t:  MsgGetHeaders,
		stop:   [32]byte(msg.HashStop),
		hashes: make([][32]byte, len(msg.BlockLocatorHashes)),
	}

	for i, hash := range msg.BlockLocatorHashes {
		record.hashes[i] = [32]byte(hash)
	}

	return record
}
