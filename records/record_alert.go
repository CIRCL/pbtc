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
	"encoding/base64"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type AlertRecord struct {
	Record

	version    int32
	relayUntil int64
	expiration int64
	id         int32
	cancel     int32
	minVer     int32
	maxVer     int32
	priority   int32
	setCancel  []int32
	setSubVer  []string
	comment    string
	statusBar  string
	reserved   string
}

func NewAlertRecord(msg *wire.MsgAlert, ra *net.TCPAddr,
	la *net.TCPAddr) *AlertRecord {
	record := &AlertRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		version:    msg.Payload.Version,
		relayUntil: msg.Payload.RelayUntil,
		expiration: msg.Payload.Expiration,
		id:         msg.Payload.ID,
		cancel:     msg.Payload.Cancel,
		minVer:     msg.Payload.MinVer,
		maxVer:     msg.Payload.MaxVer,
		priority:   msg.Payload.Priority,
		setCancel:  msg.Payload.SetCancel,
		setSubVer:  msg.Payload.SetSubVer,
		comment:    msg.Payload.Comment,
		statusBar:  msg.Payload.StatusBar,
		reserved:   msg.Payload.Reserved,
	}

	return record
}

func (ar *AlertRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(ar.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(ar.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.version), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.relayUntil), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.expiration), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.id), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.cancel), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.minVer), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.maxVer), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(ar.priority), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(ar.setCancel)), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(ar.setSubVer)), 10))

	buf.WriteString(Delimiter2)
	for i, cancel := range ar.setCancel {
		if i != 0 {
			buf.WriteString(Delimiter3)
		}

		buf.WriteString(strconv.FormatInt(int64(cancel), 10))
	}

	buf.WriteString(Delimiter2)
	for i, subver := range ar.setSubVer {
		if i != 0 {
			buf.WriteString(Delimiter3)
		}

		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(subver)))
	}

	buf.WriteString(Delimiter2)
	buf.WriteString(base64.StdEncoding.EncodeToString([]byte(ar.comment)))

	buf.WriteString(Delimiter2)
	buf.WriteString(base64.StdEncoding.EncodeToString([]byte(ar.statusBar)))

	buf.WriteString(Delimiter2)
	buf.WriteString(base64.StdEncoding.EncodeToString([]byte(ar.reserved)))

	return buf.String()
}
