package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/records"
)

type Base58Filter struct {
	log    adaptor.Log
	config []string
	next   []adaptor.Processor
}

func NewBase58(options ...func(*Base58Filter)) (*Base58Filter, error) {
	filter := &Base58Filter{}

	for _, option := range options {
		option(filter)
	}

	return filter, nil
}

func SetBase58s(base58s ...string) func(*Base58Filter) {
	return func(filter *Base58Filter) {
		filter.config = base58s
	}
}

func SetNextBase58(processors ...adaptor.Processor) func(*Base58Filter) {
	return func(filter *Base58Filter) {
		filter.next = processors
	}
}

func (filter *Base58Filter) Process(record adaptor.Record) {
	tx, ok := record.(*records.TransactionRecord)
	if !ok {
		return
	}

	for _, base58 := range filter.config {
		if tx.HasAddress(base58) {
			for _, processor := range filter.next {
				processor.Process(tx)
			}

			return
		}
	}
}
