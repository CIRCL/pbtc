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
	"time"

	"github.com/btcsuite/btcd/wire"
)

type TransactionRecord struct {
	Record

	details *DetailsRecord
}

func NewTransactionRecord(msg *wire.MsgTx, ra *net.TCPAddr,
	la *net.TCPAddr) *TransactionRecord {
	record := &TransactionRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		details: NewDetailsRecord(msg),
	}

	return record
}

func (tr *TransactionRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(tr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(tr.details.String())

	return buf.String()
}

func (tr *TransactionRecord) HasAddress(addr string) bool {
	for _, out := range tr.details.outs {
		for _, a := range out.addrs {
			if a.EncodeAddress() == addr {
				return true
			}
		}
	}

	return false
}
