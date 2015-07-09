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

// CommandFilter represents a filter that will only forward messages that fall
// under the list of defined commands.
type CommandFilter struct {
	Processor

	wg      *sync.WaitGroup
	sig     chan struct{}
	recordQ chan adaptor.Record
	config  map[string]bool
}

// NewCommand returs a new filter that will filter all messages for a list
// of defined commands. The list of commands and the processors to forward
// the records to are passed as parameters.
func NewCommandFilter(options ...func(adaptor.Processor)) (*CommandFilter, error) {
	filter := &CommandFilter{
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

// SetCommands can be passed as a parameter to NewCommand to set the list of
// commands that we want to let through our filter. If no list is provided,
// all messages will be filtered out.
func SetCommands(cmds ...string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		filter, ok := pro.(*CommandFilter)
		if !ok {
			return
		}

		for _, cmd := range cmds {
			filter.config[cmd] = true
		}
	}
}

func (filter *CommandFilter) Start() {
	filter.log.Info("[FC] Start: begin")

	filter.wg.Add(1)
	go filter.goProcess()

	filter.log.Info("[FC] Start: completed")
}

func (filter *CommandFilter) Stop() {
	filter.log.Info("[FC] Stop: begin")

	close(filter.sig)
	filter.wg.Wait()

	filter.log.Info("[FC] Stop: completed")
}

// Process adds one messages to the filter for processing and forwarding.
func (filter *CommandFilter) Process(record adaptor.Record) {
	filter.log.Debug("[FC] Process: %v", record.Command())

	filter.recordQ <- record
}

// goProcess has to be launched as a go routine.
func (filter *CommandFilter) goProcess() {
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
func (filter *CommandFilter) valid(record adaptor.Record) bool {
	return filter.config[record.Command()]
}

// forward will send the message to all processors following this filter.
func (filter *CommandFilter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
