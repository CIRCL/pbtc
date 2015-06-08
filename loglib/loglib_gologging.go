package loglib

import (
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
	consoleFormat  string
	consoleLevel   logging.Level
	fileEnabled    bool
	fileFormat     string
	fileLevel      logging.Level
	filePath       string
	file           *os.File
	levels         map[string]logging.Level
	backends       []logging.Backend
}

// NewGologging returns a new Gologging log manager, initialized with the given
// options and ready to return logs for the various modules.
func NewGologging(options ...func(log *GologgingLogger)) (*GologgingLogger,
	error) {
	logr := &GologgingLogger{
		consoleEnabled: false,
		consoleFormat: "%{color}%{time} %{level} %{shortfile} %{message}" +
			"%{color:reset}",
		consoleLevel: logging.INFO,
		fileEnabled:  false,
		fileFormat:   "%{time} %{level} %{shortfile} %{message}",
		fileLevel:    logging.INFO,
		filePath:     "pbtc.log",
		levels:       make(map[string]logging.Level),
		backends:     make([]logging.Backend, 0, 2),
	}

	for _, option := range options {
		option(logr)
	}

	if logr.consoleEnabled {
		cFormatter, err := logging.NewStringFormatter(logr.consoleFormat)
		if err != nil {
			return nil, err
		}

		cBackend := logging.NewLogBackend(os.Stderr, "", 0)
		cFormatted := logging.NewBackendFormatter(cBackend, cFormatter)
		cLeveled := logging.AddModuleLevel(cFormatted)
		cLeveled.SetLevel(logr.consoleLevel, "")
		logr.backends = append(logr.backends, cLeveled)
	}

	if logr.fileEnabled {
		file, err := os.Create(logr.filePath)
		if err != nil {
			return nil, err
		}

		fFormatter, err := logging.NewStringFormatter(logr.fileFormat)
		if err != nil {
			_ = file.Close()
			return nil, err
		}

		logr.file = file
		fBackend := logging.NewLogBackend(logr.file, "", 0)
		fFormatted := logging.NewBackendFormatter(fBackend, fFormatter)
		fLeveled := logging.AddModuleLevel(fFormatted)
		fLeveled.SetLevel(logr.fileLevel, "")
		logr.backends = append(logr.backends, fLeveled)
	}

	logging.SetBackend(logr.backends...)

	return logr, nil
}

// SetLevel has to be passed as a parameter on logger construction. It sets the
// level of a certain module, described by a string, to the given log level.
func SetLevel(module string, level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.levels[module] = level
	}
}

// EnableConsole has to be passed as a parameter on logger construction. It
// enables logging to console for this logger.
func EnableConsole() func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleEnabled = true
	}
}

// SetConsoleFormat has to be passed as a parameter on logger construction. It
// defines the format to be used by Gologging to write log lines to console.
// EnableConsole has to be passed as a parameter for this option to have an
// effect.
func SetConsoleFormat(format string) func(*GologgingLogger) {
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
func EnableFile() func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileEnabled = true
	}
}

// SetFilePath has to be passed as a parameter on logger construction. It sets
// the file path (including the file name) of the default log file.
// EnableFile must be passed as a parameter for this option to have an effect.
func SetFilePath(path string) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.filePath = path
	}
}

// SetFileFormat has to be passed as a parameter on logger construction. It
// defines the format to be used by Gologging to write log lines to a file.
// EnableFile must be passed as parameter for this option to have an effect.
func SetFileFormat(format string) func(*GologgingLogger) {
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

// Init is used to initialize the gologging library.
func (logr *GologgingLogger) Init() {
}

// Close is used to clean up after usage.
func (logr *GologgingLogger) Close() {
	_ = logr.file.Close()
}

// GetLog returns the log for a module identified with a certain string.
func (logr *GologgingLogger) GetLog(module string) adaptor.Log {
	return logging.MustGetLogger(module)
}
