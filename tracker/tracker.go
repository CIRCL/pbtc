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

package tracker

import (
	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
)

type Tracker struct {
	blocks *parmap.ParMap
	txs    *parmap.ParMap
	log    adaptor.Log
}

func New(options ...func(*Tracker)) (*Tracker, error) {
	tracker := &Tracker{
		blocks: parmap.New(),
		txs:    parmap.New(),
	}

	return tracker, nil
}

func SetLog(log adaptor.Log) func(*Tracker) {
	return func(tracker *Tracker) {
		tracker.log = log
	}
}

func (tracker *Tracker) Start() {
}

func (tracker *Tracker) Stop() {
}

func (tracker *Tracker) AddTx(hash wire.ShaHash) {
	tracker.txs.Insert(hash)
}

func (tracker *Tracker) KnowsTx(hash wire.ShaHash) bool {
	return tracker.txs.Has(hash)
}

func (tracker *Tracker) AddBlock(hash wire.ShaHash) {
	tracker.blocks.Insert(hash)
}

func (tracker *Tracker) KnowsBlock(hash wire.ShaHash) bool {
	return tracker.blocks.Has(hash)
}
