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
	"net"
	"time"

	"github.com/btcsuite/btcd/txscript"
)

const (
	Version = "PBTC LOG VERSION 1.0"
)

const (
	Delimiter1 = "|"
	Delimiter2 = ","
	Delimiter3 = "|"
)

func ParseClass(class uint8) string {
	newclass := txscript.ScriptClass(class)

	switch newclass {
	case txscript.NonStandardTy:
		return "nonstandard"

	case txscript.PubKeyTy:
		return "pubkey"

	case txscript.PubKeyHashTy:
		return "pubkeyhash"

	case txscript.ScriptHashTy:
		return "scripthash"

	case txscript.MultiSigTy:
		return "multisig"

	case txscript.NullDataTy:
		return "nulldata"

	default:
		return "invalid"
	}
}

type Record struct {
	stamp time.Time
	la    *net.TCPAddr
	ra    *net.TCPAddr
	cmd   string
}

func (r *Record) Timestamp() time.Time {
	return r.stamp
}

func (r *Record) RemoteAddress() *net.TCPAddr {
	return r.ra
}

func (r *Record) LocalAddress() *net.TCPAddr {
	return r.la
}

func (r *Record) Command() string {
	return r.cmd
}
