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

package processor

import (
	"sync"

	"github.com/CIRCL/pbtc/adaptor"
)

// DummyFilter is a placeholder filter that forwards all messages.
type DummyFilter struct {
	Processor

	wg      *sync.WaitGroup
	sig     chan struct{}
	recordQ chan adaptor.Record
}

// NewDummy creates a new DummyFilter that will forward all messages.
func NewDummy(options ...func(adaptor.Processor)) (*DummyFilter, error) {
	filter := &DummyFilter{
		wg:      &sync.WaitGroup{},
		sig:     make(chan struct{}),
		recordQ: make(chan adaptor.Record, 1),
	}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

func (filter *DummyFilter) Start() {
	filter.wg.Add(1)
	go filter.goProcess()
}

func (filter *DummyFilter) Stop() {
	close(filter.sig)
	filter.wg.Wait()
}

// Process will add a new record to the queue of the dummy filter, which will
// in turn be forwarded to the following processors.
func (filter *DummyFilter) Process(record adaptor.Record) {
	filter.recordQ <- record
}

// goProcess has to be called as a go routine. It will process and forward
// all messages in the record queue.
func (filter *DummyFilter) goProcess() {
	defer filter.wg.Done()

ProcessLoop:
	for {
		select {
		case _, ok := <-filter.sig:
			if !ok {
				break ProcessLoop
			}

		case record := <-filter.recordQ:
			if filter.valid(record) {
				filter.forward(record)
			}
		}
	}
}

// valid for dummy filter simply returns true for every record
func (filter *DummyFilter) valid(record adaptor.Record) bool {
	return true
}

// forward will send the message to the following processors for processing.
func (filter *DummyFilter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
