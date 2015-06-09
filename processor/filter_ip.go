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