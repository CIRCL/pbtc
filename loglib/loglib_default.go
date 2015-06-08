package loglib

import (
	"github.com/CIRCL/pbtc/adaptor"
)

var logr adaptor.Loglib
var err error

func init() {
	gologr, goerr := NewGologging(EnableConsole())
	logr, err = gologr, goerr
}

// New returns the default logger for the package. Use this to define default
// settings and library.
func New() (adaptor.Loglib, error) {
	return logr, err
}
