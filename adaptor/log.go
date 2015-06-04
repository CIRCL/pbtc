package adaptor

// Log defines a common interface used to inject logging into other structures.
// It makes them agnostic of the logging library, while providing signatures
// that are standard for most logs.
type Log interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Notice(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
	Critical(format string, args ...interface{})
}
