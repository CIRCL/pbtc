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
		lgr, ok := pro.(adaptor.Logger)
		if !ok {
			return
		}

		lgr.SetLog(log)
	}
}

func SetNext(next ...adaptor.Processor) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		pro.SetNext(next...)
	}
}

type Processor struct {
	log  adaptor.Log
	next []adaptor.Processor
}

func (pro *Processor) SetNext(next ...adaptor.Processor) {
	pro.next = next
}

func (pro *Processor) SetLog(log adaptor.Log) {
	pro.log = log
}
