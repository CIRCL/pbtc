package recorder

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type FilterAddRecord struct{}

type FilterClearRecord struct{}

type FilterLoadRecord struct{}

type MerkleBlockRecord struct{}

func NewFilterAddRecord(msg *wire.MsgFilterAdd, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterAddRecord {
	return &FilterAddRecord{}
}

func NewFilterClearRecord(msg *wire.MsgFilterClear, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterClearRecord {
	return &FilterClearRecord{}
}

func NewFilterLoadRecord(msg *wire.MsgFilterLoad, ra *net.TCPAddr,
	la *net.TCPAddr) *FilterLoadRecord {
	return &FilterLoadRecord{}
}

func NewMerkleBlockRecord(msg *wire.MsgMerkleBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *MerkleBlockRecord {
	return &MerkleBlockRecord{}
}
