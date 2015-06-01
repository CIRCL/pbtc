package writer

import (
	"github.com/CIRCL/pbtc/adaptor"
)

func New() (adaptor.Writer, error) {
	return NewFile()
}
