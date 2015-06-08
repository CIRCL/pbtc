package processor

import (
	"github.com/CIRCL/pbtc/adaptor"
)

// New returns a new default filter.
func New() (adaptor.Processor, error) {
	return NewDummy()
}

func SetLog(log adaptor.Log) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		pro.SetLog(log)
	}
}

func SetNext(next ...adaptor.Processor) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		pro.SetNext(next...)
	}
}
