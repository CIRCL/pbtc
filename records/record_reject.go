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
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type RejectRecord struct {
	Record

	code   uint8
	reject string
	hash   []byte
	reason string
}

func NewRejectRecord(msg *wire.MsgReject, ra *net.TCPAddr,
	la *net.TCPAddr) *RejectRecord {
	record := &RejectRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		code:   uint8(msg.Code),
		reject: msg.Cmd,
		hash:   msg.Hash.Bytes(),
		reason: msg.Reason,
	}

	return record
}

func (rr *RejectRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(rr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(rr.code), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reject)
	buf.WriteString(Delimiter1)
	buf.WriteString(hex.EncodeToString(rr.hash))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reason)

	return buf.String()
}
