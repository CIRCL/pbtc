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
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
	Record

	hdr     *HeaderRecord
	details []*DetailsRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		hdr:     NewHeaderRecord(&msg.Header),
		details: make([]*DetailsRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.details[i] = NewDetailsRecord(tx)
	}

	return record
}

func (br *BlockRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(br.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(br.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(br.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(br.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(br.hdr.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(br.details)), 10))

	buf.WriteString(Delimiter1)
	for _, tx := range br.details {
		buf.WriteString(Delimiter2)
		buf.WriteString(tx.String())
	}

	return buf.String()
}
