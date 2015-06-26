package adaptor

// Logger represents a higher level wrapper around a logging library, allowing
// us to do configuration and setup of different logging modules. It makes our
// initialization code independent from the logging library used.
type Logger interface {
	Start()
	Stop()
	GetLog(module string) Log
}
