package recorder

import (
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeaderRecord struct {
	hash    [32]byte
	version uint32
	prev    [32]byte
	root    [32]byte
	rstamp  time.Time
	bits    uint32
	nonce   uint32
}

func NewHeaderRecord(hdr *wire.BlockHeader) *HeaderRecord {
	record := &HeaderRecord{
		hash:    hdr.BlockSha(),
		version: hdr.Version,
		prev:    hdr.PrevBlock,
		root:    hdr.MerkleRoot,
		rstamp:  hdr.Timstamp,
		bits:    hdr.Bits,
		nonce:   hdr.Nonce,
	}

	return record
}
