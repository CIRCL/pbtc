package adaptor

// Log defines a common interface to inject logging into most of our modules.
type Log interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Notice(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
	Critical(format string, args ...interface{})
}
