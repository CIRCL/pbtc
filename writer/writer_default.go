package writer

import (
	"github.com/CIRCL/pbtc/adaptor"
)

func New() (adaptor.Processor, error) {
	return NewFile()
}