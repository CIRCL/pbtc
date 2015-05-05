package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type HeaderRecord struct {
	version    int32
	hash       []byte
	prev       []byte
	root       []byte
	mined      time.Time
	difficulty uint32
	nonce      uint32
}

func NewHeaderRecord(hdr *wire.BlockHeader) *HeaderRecord {
	hash := hdr.BlockSha()

	record := &HeaderRecord{
		version:    hdr.Version,
		hash:       hash.Bytes(),
		prev:       hdr.PrevBlock.Bytes(),
		root:       hdr.MerkleRoot.Bytes(),
		mined:      hdr.Timestamp,
		difficulty: hdr.Bits,
		nonce:      hdr.Nonce,
	}

	return record
}

func (hr *HeaderRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatInt(int64(hr.version), 10))
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(hr.hash))
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(hr.prev))
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(hr.root))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(hr.mined.Unix(), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(uint64(hr.difficulty), 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(uint64(hr.nonce), 10))

	return buf.String()
}

func (hr *HeaderRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, hr.version)
	binary.Write(buf, binary.LittleEndian, hr.hash)
	binary.Write(buf, binary.LittleEndian, hr.prev)
	binary.Write(buf, binary.LittleEndian, hr.root)
	binary.Write(buf, binary.LittleEndian, hr.mined.Unix())
	binary.Write(buf, binary.LittleEndian, hr.difficulty)
	binary.Write(buf, binary.LittleEndian, hr.nonce)

	return buf.Bytes()
}
