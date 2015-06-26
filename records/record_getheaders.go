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

type GetHeadersRecord struct {
	Record

	stop   [32]byte
	hashes [][32]byte
}

func NewGetHeadersRecord(msg *wire.MsgGetHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *GetHeadersRecord {
	record := &GetHeadersRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		stop:   msg.HashStop,
		hashes: make([][32]byte, len(msg.BlockLocatorHashes)),
	}

	for i, hash := range msg.BlockLocatorHashes {
		record.hashes[i] = *hash
	}

	return record
}

func (gr *GetHeadersRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(hex.EncodeToString(gr.stop[:]))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(gr.hashes)), 10))

	for _, hash := range gr.hashes {
		buf.WriteString(Delimiter2)
		buf.WriteString(hex.EncodeToString(hash[:]))
	}

	return buf.String()
}
