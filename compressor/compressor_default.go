package compressor

import (
	"github.com/CIRCL/pbtc/adaptor"
)

// New is a shortcut to create a default compressor. If you want to change the
// type and options of the default compressor, this is where you do it.
func New() adaptor.Compressor {
	return NewDummy()
}
