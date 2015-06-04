package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
)

// New returns a new default filter.
func New() (adaptor.Processor, error) {
	return NewDummy()
}
