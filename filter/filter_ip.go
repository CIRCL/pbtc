package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
)

type IPFilter struct {
	config map[string]bool
	next   []adaptor.Processor
}

func NewIP(options ...func(*IPFilter)) (*IPFilter, error) {
	filter := &IPFilter{
		config: make(map[string]bool),
	}

	return filter, nil
}

func SetIPs(ips ...string) func(*IPFilter) {
	return func(filter *IPFilter) {
		for _, ip := range ips {
			filter.config[ip] = true
		}
	}
}

func SetNextIP(processors ...adaptor.Processor) func(*IPFilter) {
	return func(filter *IPFilter) {
		filter.next = processors
	}
}

func (filter *IPFilter) Process(record adaptor.Record) {
	if !filter.config[record.Address().IP.String()] {
		return
	}

	for _, processor := range filter.next {
		processor.Process(record)
	}
}
