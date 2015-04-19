package domain

import (
	"os"

	"github.com/op/go-logging"
)

type LoggerStub struct {
	logger Logger
}

func NewLoggerStub(options ...func(*LoggerStub)) *LoggerStub {
	logging.SetLevel(logging.DEBUG, "")
	logging.SetBackend(logging.NewLogBackend(os.Stderr, "", 0))

	log := &LoggerStub{
		logger: logging.MustGetLogger(""),
	}

	return log
}

func (log *LoggerStub) Debug(format string, args ...interface{}) {
	log.logger.Debug(format, args...)
}

func (log *LoggerStub) Info(format string, args ...interface{}) {
	log.logger.Info(format, args...)
}

func (log *LoggerStub) Notice(format string, args ...interface{}) {
	log.logger.Notice(format, args...)
}

func (log *LoggerStub) Warning(format string, args ...interface{}) {
	log.logger.Warning(format, args...)
}

func (log *LoggerStub) Error(format string, args ...interface{}) {
	log.logger.Error(format, args...)
}

func (log *LoggerStub) Critical(format string, args ...interface{}) {
	log.logger.Critical(format, args...)
}
