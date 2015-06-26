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

// IPFilter is a filter to forward only messages that come from a peer whose
// remote address is in the given list of IP addresses.
type IPFilter struct {
	Processor

	wg      *sync.WaitGroup
	sig     chan struct{}
	recordQ chan adaptor.Record
	config  map[string]bool
}

// NewIP creates a new IP filter that will only forward messages coming from
// a given set of IP addresses.
func NewIPFilter(options ...func(adaptor.Processor)) (*IPFilter, error) {
	filter := &IPFilter{
		wg:      &sync.WaitGroup{},
		sig:     make(chan struct{}),
		recordQ: make(chan adaptor.Record, 1),
		config:  make(map[string]bool),
	}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

// SetIPs can be passed as a parameter to NewIP to set the list of IP addresses
// to filter for. If no list is provided, all messages are filtered out.
func SetIPs(ips ...string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		filter, ok := pro.(*IPFilter)
		if !ok {
			return
		}

		for _, ip := range ips {
			filter.config[ip] = true
		}
	}
}

func (filter *IPFilter) Start() {
	filter.wg.Add(1)
	go filter.goProcess()
}

func (filter *IPFilter) Stop() {
	close(filter.sig)
	filter.wg.Wait()
}

// Process will add a record to the queue of records to be processed.
func (filter *IPFilter) Process(record adaptor.Record) {
	filter.recordQ <- record
}

// goProcess has to be launched as a go routine.
func (filter *IPFilter) goProcess() {
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
func (filter *IPFilter) valid(record adaptor.Record) bool {
	return filter.config[record.RemoteAddress().IP.String()]
}

// forward will send the message to the following processors for processing.
func (filter *IPFilter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
