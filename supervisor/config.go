package supervisor

type Config struct {
	Logging struct {
		Console_enabled bool
		Console_format  string
		Console_level   string
		File_enabled    bool
		File_format     string
		File_level      string
		File_path       string
	}

	Repository map[string]*struct {
		Log_level   string
		Seeds_list  []string
		Seeds_port  uint16
		Backup_rate uint32
		Backup_path string
		Node_limit  uint32
	}

	Tracker map[string]*struct {
		Log_level string
	}

	Server map[string]*struct {
		Log_level    string
		Address_list []string
	}

	Manager map[string]*struct {
		Log_level        string
		Protocol_magic   uint32
		Protocol_version uint32
		Connection_rate  uint32
		Connection_limit uint32
	}

	Processor map[string]*struct {
		Log_level string
	}
}
