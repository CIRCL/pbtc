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

	"github.com/btcsuite/btcd/wire"
)

type ItemRecord struct {
	category uint8
	hash     [32]byte
}

func NewItemRecord(vec *wire.InvVect) *ItemRecord {
	ir := &ItemRecord{
		category: uint8(vec.Type),
		hash:     vec.Hash,
	}

	return ir
}

func (ir *ItemRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatUint(uint64(ir.category), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(hex.EncodeToString(ir.hash[:]))

	return buf.String()
}
