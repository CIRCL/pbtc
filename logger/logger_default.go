package logger

import (
	"github.com/CIRCL/pbtc/adaptor"
)

var logr adaptor.Logger
var err error

func init() {
	gologr, goerr := NewGologging(EnableConsole())
	logr, err = gologr, goerr
}

// New returns the default logger for the package. Use this to define default
// settings and library.
func New() (adaptor.Logger, error) {
	return logr, err
}
