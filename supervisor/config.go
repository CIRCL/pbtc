package supervisor

import (
	"github.com/btcsuite/btcd/wire"
	"time"
)

type Config struct {
	Repository map[string]*struct {
		Seeds_list  []string
		Seeds_port  int
		Backup_path string
		Backup_rate time.Duration
		Node_limit  int
	}

	Tracker map[string]*struct {
	}

	Server map[string]*struct {
		Address_list []string

		Manager string
	}

	Manager map[string]*struct {
		Protocol_type    uint32
		Protocol_version uint32
		Conn_rate        time.Duration
		Conn_limit       int

		Repository string
		Tracker    string
		Processor  []string
	}

	Processor map[string]*struct {
		Type string
		Next []string
	}

	Compressor map[string]*struct {
		Type string
	}
}
