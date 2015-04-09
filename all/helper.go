package all

type LogHelper struct {
	agent  *LogAgent
	prefix string
}

// NewLogHelper initializes a new log helper to receive the input from a module. It will add
// a prefix to all the logging messags.
func NewLogHelper(agent *LogAgent, prefix string) *LogHelper {
	helper := &LogHelper{
		agent:  agent,
		prefix: prefix,
	}

	return helper
}

// Log logs the parameters provided joining them together with spaces.
func (helper *LogHelper) Log(severity uint32, v ...interface{}) {
	args := make([]interface{}, len(v)+1)
	args[0] = helper.prefix
	copy(args[1:], v)
	helper.agent.log(severity, args...)
}

// Logf logs the given format string with keywords replaced by the remaining parameters.
func (helper *LogHelper) Logf(severity uint32, format string, v ...interface{}) {
	helper.agent.logf(severity, helper.prefix+format, v...)
}

// Logln logs the parameters provided joining them together with spaces and adding a new line.
func (helper *LogHelper) Logln(severity uint32, v ...interface{}) {
	args := make([]interface{}, len(v)+1)
	args[0] = helper.prefix
	copy(args[1:], v)
	helper.agent.logln(severity, args...)
}
