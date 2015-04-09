package all

import (
	"log"
)

const (
	LogTrace = iota
	LogDebug
	LogInfo
	LogWarning
	LogError
	LogFatal
)

// LogAgent manages the logging for our application.
type LogAgent struct {
	output map[*log.Logger]uint32
	input  map[string]*LogHelper
}

var logAgent *LogAgent

func init() {
	logAgent = GetLogAgent()
}

// GetLogHelper returns the log helper for the given prefix.
func GetLogHelper(prefix string) *LogHelper {
	return logAgent.getInput(prefix)
}

// NewLogAgent initializes a new log agent with standard configuration.
func GetLogAgent() *LogAgent {
	if logAgent == nil {
		logAgent = &LogAgent{
			output: make(map[*log.Logger]uint32),
			input:  make(map[string]*LogHelper),
		}
	}

	return logAgent
}

// AddOutput adds a logger as output at the given level.
func (agent *LogAgent) AddOutput(logger *log.Logger, level uint32) {
	_, ok := agent.output[logger]
	if ok {
		return
	}

	agent.output[logger] = level
}

// RemoveOutput removes a logger as output.
func (agent *LogAgent) RemoveOutput(logger *log.Logger) {
	_, ok := agent.output[logger]
	if !ok {
		return
	}

	delete(agent.output, logger)
}

// AddInput adds any class as an input and returns a logging helper or it to write to.
func (agent *LogAgent) getInput(prefix string) *LogHelper {
	helper, ok := agent.input[prefix]
	if ok {
		return helper
	}

	helper = NewLogHelper(agent, prefix)
	agent.input[prefix] = helper
	return helper
}

// SetLevel sets the logging level for the given logger.
func (agent *LogAgent) SetLevel(logger *log.Logger, level uint32) {
	_, ok := agent.output[logger]
	if !ok {
		return
	}

	agent.output[logger] = level
}

// Log logs the parameters provided joining them together with spaces.
func (agent *LogAgent) log(severity uint32, v ...interface{}) {
	for logger, level := range agent.output {
		if level <= severity {
			logger.Print(v...)
		}
	}
}

// Logf logs the given format string with keywords replaced by the remaining parameters.
func (agent *LogAgent) logf(severity uint32, format string, v ...interface{}) {
	for logger, level := range agent.output {
		if level <= severity {
			logger.Printf(format, v...)
		}
	}
}

// Logln logs the parameters provided joining them together with spaces and adding a new line.
func (agent *LogAgent) logln(severity uint32, v ...interface{}) {
	for logger, level := range agent.output {
		if level <= severity {
			logger.Println(v...)
		}
	}
}
