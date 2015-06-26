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

	"github.com/CIRCL/pbtc/util"
)

type EntryRecord struct {
	addr     *net.TCPAddr
	stamp    time.Time
	services uint64
}

func NewEntryRecord(na *wire.NetAddress) *EntryRecord {
	record := &EntryRecord{
		addr:     util.ParseNetAddress(na),
		stamp:    na.Timestamp,
		services: uint64(na.Services),
	}

	return record
}

func (er *EntryRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(strconv.FormatInt(er.stamp.Unix(), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(er.services, 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(er.addr.String())

	return buf.String()
}
