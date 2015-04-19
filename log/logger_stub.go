package log

type LoggerStub struct {
}

func (log *LoggerStub) Debug(format string, args ...interface{}) {

}

func (log *LoggerStub) Info(format string, args ...interface{}) {

}

func (log *LoggerStub) Notice(format string, args ...interface{}) {

}

func (log *LoggerStub) Warning(format string, args ...interface{}) {

}

func (log *LoggerStub) Error(format string, args ...interface{}) {

}

func (log *LoggerStub) Critical(format string, args ...interface{}) {

}
