package logger

import (
	"os"

	"github.com/op/go-logging"
)

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

func New(options ...func(log *GologgingLogger)) (*GologgingLogger, error) {
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

func SetLevel(module string, level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.levels[module] = level
	}
}

func EnableConsole() func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleEnabled = true
	}
}

func EnableFile() func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileEnabled = true
	}
}

// SetConsoleFormat sets the format to be used for standard error logging.
func SetConsoleFormat(format string) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleFormat = format
	}
}

// SetConsoleLevel sets the level for loggng to standard error.
func SetConsoleLevel(level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.consoleLevel = level
	}
}

// SetFilePath sets the path and name for the logging file.
func SetFilePath(path string) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.filePath = path
	}
}

// SetFileFormat sets the format to be used for file logging.
func SetFileFormat(format string) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileFormat = format
	}
}

// SetFileLevel sets the level for logging to file.
func SetFileLevel(level logging.Level) func(*GologgingLogger) {
	return func(logr *GologgingLogger) {
		logr.fileLevel = level
	}
}

func (logr *GologgingLogger) GetLog(module string) *logging.Logger {
	return logging.MustGetLogger(module)
}

func (logr *GologgingLogger) Stop() {
	_ = logr.file.Close()
}
