package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
)

type CommandFilter struct {
	log    adaptor.Log
	config map[string]bool
	next   []adaptor.Processor
}

func NewCommand(options ...func(*CommandFilter)) (*CommandFilter, error) {
	filter := &CommandFilter{
		config: make(map[string]bool),
	}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

// SetLogger injects the logger to be used for logging.
func SetLogCommand(log adaptor.Log) func(*CommandFilter) {
	return func(filter *CommandFilter) {
		filter.log = log
	}
}

// SetCommands defines a filter on the type of message. If defined, only
// messages of the given types will be forwarded.
func SetCommands(cmds ...string) func(*CommandFilter) {
	return func(filter *CommandFilter) {
		for _, cmd := range cmds {
			filter.config[cmd] = true
		}
	}
}

func SetNextCommand(processors ...adaptor.Processor) func(*CommandFilter) {
	return func(filter *CommandFilter) {
		filter.next = processors
	}
}

func (filter *CommandFilter) Process(record adaptor.Record) {
	if !filter.config[record.Cmd()] {
		return
	}

	for _, processor := range filter.next {
		processor.Process(record)
	}
}
