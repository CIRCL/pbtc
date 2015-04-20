package logger

import (
	"os"

	"github.com/op/go-logging"
)

type MultiLogger struct {
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

func New(options ...func(*MultiLogger)) (Logger, error) {

	log := &MultiLogger{
		filePath:      "pbtc.log",
		fileFormat:    "%{time} %{level} %{shortfile} %{message}",
		fileLevel:     logging.INFO,
		consoleFormat: "%{color}%{time} %{level} %{shortfile} %{message}%{color:reset}",
		consoleLevel:  logging.INFO,
		backends:      make([]logging.Backend, 0, 2),
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

func EnableConsole() func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.consoleEnabled = true
	}
}

func SetConsoleFormat(format string) func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.consoleFormat = format
	}
}

func SetConsoleLevel(level logging.Level) func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.consoleLevel = level
	}
}

func EnableFile() func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.fileEnabled = true
	}
}

func SetFilePath(path string) func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.filePath = path
	}
}

func SetFileFormat(format string) func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.fileFormat = format
	}
}

func SetFileLevel(level logging.Level) func(*MultiLogger) {
	return func(log *MultiLogger) {
		log.fileLevel = level
	}
}
