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

	filter.wg.Add(1)
	go filter.goProcess()

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

// Process adds one messages to the filter for processing and forwarding.
func (filter *CommandFilter) Process(record adaptor.Record) {
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
	return filter.config[record.Cmd()]
}

// forward will send the message to all processors following this filter.
func (filter *CommandFilter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
