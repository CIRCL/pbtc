package processor

import (
	"sync"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/records"
)

// Base58Filter is a filter which only forwards transactions if they contain
// an output to one of the given Bitcoin addresses.
type Base58Filter struct {
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
func NewBase58Filter(options ...func(adaptor.Processor)) (*Base58Filter, error) {
	filter := &Base58Filter{
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

// SetBase58s can be passed as parameter to NewBase58 in order to define the
// list of Bitcoin addresses we want to filter transactions for. If this
// parameter is not passed, no records will be forwarded.
func SetBase58s(base58s ...string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		filter, ok := pro.(*Base58Filter)
		if !ok {
			return
		}

		filter.config = base58s
	}
}

// Process adds one messages to the filter for processing and forwarding.
func (filter *Base58Filter) Process(record adaptor.Record) {
	filter.recordQ <- record
}

// Close will end the filter and wait for the go routine to quit.
func (filter *Base58Filter) Close() {
	close(filter.sig)
	filter.wg.Wait()
}

// goProcess is to be launched as a go routine. It reads the records added to
// the queue and forwards valid records to the next set of processors.
func (filter *Base58Filter) goProcess() {
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
func (filter *Base58Filter) valid(record adaptor.Record) bool {
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
func (filter *Base58Filter) forward(record adaptor.Record) {
	for _, processor := range filter.next {
		processor.Process(record)
	}
}
