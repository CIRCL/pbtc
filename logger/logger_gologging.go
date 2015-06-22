package logger

import (
	"errors"
	"os"

	"github.com/op/go-logging"

	"github.com/CIRCL/pbtc/adaptor"
)

// GologgingLogger is a wrapper around the Go-logging library. It uses package
// state to initialize loggers, which makes it difficult to use it directly
// without breaking the loose coupling principle. Using this wrapper allows us
// to change the library in the future without having to rewrite other packages.
type GologgingLogger struct {
	consoleEnabled bool
	consoleFormat  logging.Formatter
	consoleLevel   logging.Level
	fileEnabled    bool
	fileFormat     logging.Formatter
	fileLevel      logging.Level
	file           *os.File
	backends       []logging.Backend
}

func ParseLevel(level string) (logging.Level, error) {
	switch level {
	case "DEBUG":
		return logging.DEBUG, nil

	case "INFO":
		return logging.INFO, nil

	case "NOTICE":
		return logging.NOTICE, nil

	case "WARNING":
		return logging.WARNING, nil

	case "ERROR":
		return logging.ERROR, nil

	case "CRITICAL":
		return logging.CRITICAL, nil

	default:
		return -1, errors.New("invalid logging level string")
	}
}

func ParseFormat(format string) (logging.Formatter, error) {
	return logging.NewStringFormatter(format)
}

// NewGologging returns a new Gologging log manager, initialized with the given
// options and ready to return logs for the various modules.
func NewGologging(options ...func(log *GologgingLogger)) (*GologgingLogger,
	error) {
	logr := &GologgingLogger{
		consoleEnabled: false,
		consoleFormat:  logging.MustStringFormatter("%{message}"),
		consoleLevel:   logging.CRITICAL,
		fileEnabled:    false,
		fileFormat:     logging.MustStringFormatter("%{message}"),
		fileLevel:      logging.CRITICAL,
		backends:       make([]logging.Backend, 0, 2),
	}

	for _, option := range options {
		option(logr)
	}

	if logr.consoleEnabled {
		cBackend := logging.NewLogBackend(os.Stderr, "", 0)
		cFormatted := logging.NewBackendFormatter(cBackend, logr.consoleFormat)
		cLeveled := logging.AddModuleLevel(cFormatted)
		cLeveled.SetLevel(logr.consoleLevel, "")
		logr.backends = append(logr.backends, cLeveled)
	}

	if logr.fileEnabled {
		if logr.file == nil {
			return nil, errors.New("invalid file path for logging")
		}

		fBackend := logging.NewLogBackend(logr.file, "", 0)
		fFormatted := logging.NewBackendFormatter(fBackend, logr.fileFormat)
		fLeveled := logging.AddModuleLevel(fFormatted)
		fLeveled.SetLevel(logr.fileLevel, "")
		logr.backends = append(logr.backends, fLeveled)
	}

	logging.SetBackend(logr.backends...)

	return logr, nil
}

// EnableConsole has to be passed as a parameter on logger construction. It
// enables logging to console for this logger.
func SetConsoleEnabled(enabled bool) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleEnabled = enabled
	}
}

// SetConsoleFormat has to be passed as a parameter on logger construction. It
// defines the format to be used by Gologging to write log lines to console.
// EnableConsole has to be passed as a parameter for this option to have an
// effect.
func SetConsoleFormat(format logging.Formatter) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleFormat = format
	}
}

// SetConsoleLevel has to be passed as a paramater on logger construction. It
// sets the default logging level for the console output.
// EnableConsole has to be passed as a parameter for this option to have an
// effect.
func SetConsoleLevel(level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleLevel = level
	}
}

// EnableFile has to be passed as a parameter on logger construction. It enables
// logging to a file for this logger.
func SetFileEnabled(enabled bool) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileEnabled = enabled
	}
}

// SetFilePath has to be passed as a parameter on logger construction. It sets
// the file path (including the file name) of the default log file.
// EnableFile must be passed as a parameter for this option to have an effect.
func SetFile(file *os.File) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.file = file
	}
}

// SetFileFormat has to be passed as a parameter on logger construction. It
// defines the format to be used by Gologging to write log lines to a file.
// EnableFile must be passed as parameter for this option to have an effect.
func SetFileFormat(format logging.Formatter) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileFormat = format
	}
}

// SetFileLevel has to be passed as a parameter on logger construction. It
// sets the default logging level for the file output.
// EnableFile must be passed as parameter for this option to have an effect.
func SetFileLevel(level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileLevel = level
	}
}

// Close is used to clean up after usage.
func (logr *GologgingLogger) Close() {
	_ = logr.file.Close()
}

// GetLog returns the log for a module identified with a certain string.
func (logr *GologgingLogger) GetLog(module string) adaptor.Log {
	return logging.MustGetLogger(module)
}

func (logr *GologgingLogger) SetLevel(level logging.Level, module string) {
	logging.SetLevel(level, module)
}
