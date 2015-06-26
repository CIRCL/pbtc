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
	"encoding/gob"
	"net"
	"time"
)

type node struct {
	addr          *net.TCPAddr
	numSeen       uint32
	numAttempts   uint32
	lastAttempted time.Time
	lastConnected time.Time
	lastSucceeded time.Time
}

func newNode(addr *net.TCPAddr) *node {
	n := &node{
		addr:    addr,
		numSeen: 1,
	}

	return n
}

func (node *node) String() string {
	return node.addr.String()
}

// GobEncode is required to implement the GobEncoder interface.
// It allows us to serialize the unexported fields of our nodes.
// We could also change them to exported, but as nodes are only
// handled internally in the repository, this is the better choice.
func (node *node) GobEncode() ([]byte, error) {
	buffer := &bytes.Buffer{}
	enc := gob.NewEncoder(buffer)

	err := enc.Encode(node.addr)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.numAttempts)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastAttempted)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastConnected)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastSucceeded)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// GobDecode is required to implement the GobDecoder interface.
// It allows us to deserialize the unexported fields of our nodes.
func (node *node) GobDecode(buf []byte) error {
	buffer := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&node.addr)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.numAttempts)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastAttempted)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastConnected)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastSucceeded)
	if err != nil {
		return err
	}

	return nil
}
