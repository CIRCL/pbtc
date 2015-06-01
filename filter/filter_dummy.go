package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
)

type DummyFilter struct {
	next []adaptor.Processor
}

func NewDummy(options ...func(*DummyFilter)) (*DummyFilter, error) {
	filter := &DummyFilter{}

	return filter, nil
}

func SetNextDummy(processors ...adaptor.Processor) func(*DummyFilter) {
	return func(filter *DummyFilter) {
		filter.next = processors
	}
}

func (filter *DummyFilter) Process(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
