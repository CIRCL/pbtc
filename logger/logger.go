package logger

import (
	"os"

	"github.com/op/go-logging"
)

// Logger implements our logging interface by returning an object from the
// go-logging library which is compatible with our adaptor. It is a powerful
// logging implementation that allows us to configure multiple front- and
// back-ends.
type Logger struct {
	file        *os.File
	fileEnabled bool
	filePath    string
	fileFormat  string
	fileLevel   logging.Level

	consoleEnabled bool
	consoleFormat  string
	consoleLevel   logging.Level

	backends []logging.Backend
}

// New creates a new logger with the help of the go-logging library. It supports
// a number of options to enable/disable file/console logging, set levels and
// formats.
func New(options ...func(*Logger)) (*logging.Logger, error) {

	log := &Logger{
		filePath:   "pbtc.log",
		fileFormat: "%{time} %{level} %{shortfile} %{message}",
		fileLevel:  logging.INFO,
		consoleFormat: "%{color}%{time} %{level} %{shortfile} %{message}" +
			"%{color:reset}",
		consoleLevel: logging.INFO,
		backends:     make([]logging.Backend, 0, 2),
	}

	for _, option := range options {
		option(log)
	}

	if log.consoleEnabled {
		consoleBackend := logging.NewLogBackend(os.Stderr, "", 0)

		consoleFormatter, err := logging.NewStringFormatter(log.consoleFormat)
		if err != nil {
			return nil, err
		}

		consoleFormatted := logging.NewBackendFormatter(consoleBackend, consoleFormatter)
		consoleLeveled := logging.AddModuleLevel(consoleFormatted)
		consoleLeveled.SetLevel(log.consoleLevel, "pbtc")
		log.backends = append(log.backends, consoleLeveled)
	}

	if log.fileEnabled {
		file, err := os.Create(log.filePath)
		if err != nil {
			return nil, err
		}

		log.file = file
		fileBackend := logging.NewLogBackend(log.file, "", 0)

		fileFormatter, err := logging.NewStringFormatter(log.fileFormat)
		if err != nil {
			return nil, err
		}

		fileFormatted := logging.NewBackendFormatter(fileBackend, fileFormatter)
		fileLeveled := logging.AddModuleLevel(fileFormatted)
		fileLeveled.SetLevel(log.fileLevel, "pbtc")
		log.backends = append(log.backends, fileLeveled)
	}

	logging.SetBackend(log.backends...)

	return logging.MustGetLogger("pbtc"), nil

}

// EnableConsole enables logging to standard error.
func EnableConsole() func(*Logger) {
	return func(log *Logger) {
		log.consoleEnabled = true
	}
}

// SetConsoleFormat sets the format to be used for standard error logging.
func SetConsoleFormat(format string) func(*Logger) {
	return func(log *Logger) {
		log.consoleFormat = format
	}
}

// SetConsoleLevel sets the level for loggng to standard error.
func SetConsoleLevel(level logging.Level) func(*Logger) {
	return func(log *Logger) {
		log.consoleLevel = level
	}
}

// EnableFle enables logging to a log file.
func EnableFile() func(*Logger) {
	return func(log *Logger) {
		log.fileEnabled = true
	}
}

// SetFilePath sets the path and name for the logging file.
func SetFilePath(path string) func(*Logger) {
	return func(log *Logger) {
		log.filePath = path
	}
}

// SetFileFormat sets the format to be used for file logging.
func SetFileFormat(format string) func(*Logger) {
	return func(log *Logger) {
		log.fileFormat = format
	}
}

// SetFileLevel sets the level for logging to file.
func SetFileLevel(level logging.Level) func(*Logger) {
	return func(log *Logger) {
		log.fileLevel = level
	}
}
