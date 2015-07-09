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
	"github.com/CIRCL/pbtc/records"
)

// AddressFilter is a filter which only forwards transactions if they contain
// an output to one of the given Bitcoin addresses.
type AddressFilter struct {
	Processor

	wg      *sync.WaitGroup
	sig     chan struct{}
	recordQ chan adaptor.Record
	config  []string
}

// NewBase58 creates a new filter that only forwards transactions if they
// contain one output ot one of the given Bitcoin addresses. The list of
// Bitcoin addresses and the processors to forward the transactions to are
// passed as parameters on construction.
func NewAddressFilter(options ...func(adaptor.Processor)) (*AddressFilter, error) {
	filter := &AddressFilter{
		wg:      &sync.WaitGroup{},
		sig:     make(chan struct{}),
		recordQ: make(chan adaptor.Record, 1),
	}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

// SetBase58s can be passed as parameter to NewBase58 in order to define the
// list of Bitcoin addresses we want to filter transactions for. If this
// parameter is not passed, no records will be forwarded.
func SetAddresses(addresses ...string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		filter, ok := pro.(*AddressFilter)
		if !ok {
			return
		}

		filter.config = addresses
	}
}

func (filter *AddressFilter) Start() {
	filter.log.Info("[FA] Start: begin")

	filter.wg.Add(1)
	go filter.goProcess()

	filter.log.Info("[FA] Start: completed")
}

// Close will end the filter and wait for the go routine to quit.
func (filter *AddressFilter) Stop() {
	filter.log.Info("[FA] Stop: begin")

	close(filter.sig)
	filter.wg.Wait()

	filter.log.Info("[FA] Stop: completed")
}

// Process adds one messages to the filter for processing and forwarding.
func (filter *AddressFilter) Process(record adaptor.Record) {
	filter.log.Debug("[FA] PRocess: %v", record.Command())

	filter.recordQ <- record
}

// goProcess is to be launched as a go routine. It reads the records added to
// the queue and forwards valid records to the next set of processors.
func (filter *AddressFilter) goProcess() {
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

// valid checks whether a record fulfills the criteria for forwarding.
func (filter *AddressFilter) valid(record adaptor.Record) bool {
	tx, ok := record.(*records.TransactionRecord)
	if !ok {
		return false
	}

	for _, base58 := range filter.config {
		if tx.HasAddress(base58) {
			return true
		}
	}

	return false
}

// forward will send the message to all processors following this filter.
func (filter *AddressFilter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
