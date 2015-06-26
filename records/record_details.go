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

type DetailsRecord struct {
	hash [32]byte
	ins  []*InputRecord
	outs []*OutputRecord
}

func NewDetailsRecord(msg *wire.MsgTx) *DetailsRecord {
	record := &DetailsRecord{
		hash: msg.TxSha(),
		ins:  make([]*InputRecord, len(msg.TxIn)),
		outs: make([]*OutputRecord, len(msg.TxOut)),
	}

	for i, txin := range msg.TxIn {
		record.ins[i] = NewInputRecord(txin)
	}

	for i, txout := range msg.TxOut {
		record.outs[i] = NewOutputRecord(txout)
	}

	return record
}

func (dr *DetailsRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(hex.EncodeToString(dr.hash[:]))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(len(dr.ins)), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(len(dr.outs)), 10))

	for _, input := range dr.ins {
		buf.WriteString(Delimiter2)
		buf.WriteString(input.String())
	}

	for _, output := range dr.outs {
		buf.WriteString(Delimiter2)
		buf.WriteString(output.String())
	}

	return buf.String()
}
