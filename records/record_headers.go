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

type HeadersRecord struct {
	Record

	hdrs []*HeaderRecord
}

func NewHeadersRecord(msg *wire.MsgHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *HeadersRecord {
	record := &HeadersRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		hdrs: make([]*HeaderRecord, len(msg.Headers)),
	}

	for i, hdr := range msg.Headers {
		record.hdrs[i] = NewHeaderRecord(hdr)
	}

	return record
}

func (hr *HeadersRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(hr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(hr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(hr.hdrs)), 10))

	for _, hdr := range hr.hdrs {
		buf.WriteString(Delimiter2)
		buf.WriteString(hdr.String())
	}

	return buf.String()
}
