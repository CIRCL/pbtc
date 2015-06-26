// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

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
