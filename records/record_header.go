package records

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeaderRecord struct {
	block_hash  [32]byte
	version     int32
	prev_block  [32]byte
	merkle_root [32]byte
	timestamp   time.Time
	bits        uint32
	nonce       uint32
	txn_count   uint8
}

func NewHeaderRecord(hdr *wire.BlockHeader) *HeaderRecord {
	record := &HeaderRecord{
		block_hash:  hdr.BlockSha(), // this is calculated, not sent
		version:     hdr.Version,
		prev_block:  hdr.PrevBlock,
		merkle_root: hdr.MerkleRoot,
		timestamp:   hdr.Timestamp,
		bits:        hdr.Bits,
		nonce:       hdr.Nonce,
		txn_count:   0,
	}

	return record
}

func (hr *HeaderRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(hex.EncodeToString(hr.block_hash[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(hr.version), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(hex.EncodeToString(hr.prev_block[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(hex.EncodeToString(hr.merkle_root[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(hr.timestamp.Unix(), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(hr.bits), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(hr.nonce), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(hr.txn_count), 10))

	return buf.String()
}
