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

package parmap

import (
	"fmt"
	"sync"
)

// shard is one synchronized hashmap used for a section of the global map
type shard struct {
	index map[string]fmt.Stringer
	mutex *sync.RWMutex
}

// newShard creates a new shard with an initialized mutex for synchronization
func newShard() *shard {
	shard := &shard{
		index: make(map[string]fmt.Stringer),
		mutex: &sync.RWMutex{},
	}

	return shard
}
