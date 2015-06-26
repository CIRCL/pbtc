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

package repository

import (
	"bytes"
	"net"
)

type ipRange struct {
	start net.IP
	end   net.IP
}

func newIPRange(start string, end string) *ipRange {
	r := &ipRange{
		start: net.ParseIP(start),
		end:   net.ParseIP(end),
	}

	return r
}

func (r *ipRange) includes(ip net.IP) bool {
	if bytes.Compare(r.start, ip) >= 0 && bytes.Compare(r.end, ip) <= 0 {
		return true
	}

	return false
}
