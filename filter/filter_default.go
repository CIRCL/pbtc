package filter

import (
	"github.com/CIRCL/pbtc/adaptor"
)

func New() (adaptor.Processor, error) {
	return NewBase58()
}
