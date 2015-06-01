package logger

import (
	"github.com/CIRCL/pbtc/adaptor"
)

var logr adaptor.Logger

func init() {
	golog, err := NewGologging(EnableConsole())
	if err != nil {
		panic("Could not initialize default logger")
	}

	logr = golog
}

// New returns the default logger for the package. Use this to define default
// settings and library.
func New() adaptor.Logger {
	return logr
}
