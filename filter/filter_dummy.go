package filter

import (
	"sync"

	"github.com/CIRCL/pbtc/adaptor"
)

// DummyFilter is a placeholder filter that forwards all messages.
type DummyFilter struct {
	wg      *sync.WaitGroup
	sig     chan struct{}
	recordQ chan adaptor.Record
	log     adaptor.Log
	next    []adaptor.Processor
}

// NewDummy creates a new DummyFilter that will forward all messages.
func NewDummy(options ...func(*DummyFilter)) (*DummyFilter, error) {
	filter := &DummyFilter{
		wg:      &sync.WaitGroup{},
		sig:     make(chan struct{}),
		recordQ: make(chan adaptor.Record, 1),
	}

	for _, option := range options {
		option(filter)
	}

	filter.wg.Add(1)
	go filter.goProcess()

	return filter, nil
}

// SetLogDummy can be passed as a parameter to NewDummy to set the log for
// output.
func SetLogDummy(log adaptor.Log) func(*DummyFilter) {
	return func(filter *DummyFilter) {
		filter.log = log
	}
}

// SetNextDummy can be passed as a parameter to NewDummy to set the list of
// processors that we will forward the messages to.
func SetNextDummy(processors ...adaptor.Processor) func(*DummyFilter) {
	return func(filter *DummyFilter) {
		filter.next = processors
	}
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
