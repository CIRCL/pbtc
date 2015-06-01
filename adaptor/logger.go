package adaptor

// Logger represents the wrapper around our logging library. It can return logs
// and set levels for certain module strings, which allows us to handle this
// in our own code if the library doesn't support it.
type Logger interface {
	GetLog(module string) Log
}
