package supervisor

type Config struct {
	Logger     map[string]*LoggerConfig
	Repository map[string]*RepositoryConfig
	Tracker    map[string]*TrackerConfig
	Server     map[string]*ServerConfig
	Processor  map[string]*ProcessorConfig
	Manager    map[string]*ManagerConfig
}

type LoggerConfig struct {
	Console_enabled bool
	Console_format  string
	Console_level   string
	File_enabled    bool
	File_format     string
	File_level      string
	File_path       string
}

type RepositoryConfig struct {
	Log_level   string
	Seeds_list  []string
	Seeds_port  uint16
	Backup_rate uint32
	Backup_path string
	Node_limit  uint32
}

type TrackerConfig struct {
	Log_level string
}

type ServerConfig struct {
	Log_level    string
	Address_list []string
}

type ProcessorConfig struct {
	Log_level string
}

type ManagerConfig struct {
	Log_level        string
	Protocol_magic   uint32
	Protocol_version uint32
	Connection_rate  uint32
	Connection_limit uint32
}
