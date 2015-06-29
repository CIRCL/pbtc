// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

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
	Log_level       string
	Console_enabled bool
	Console_format  string
	Console_level   string
	File_enabled    bool
	File_format     string
	File_level      string
	File_path       string
}

type RepositoryConfig struct {
	Logger      string
	Log_level   string
	Seeds_list  []string
	Seeds_port  uint16
	Backup_rate uint32
	Backup_path string
	Node_limit  uint32
}

type TrackerConfig struct {
	Logger    string
	Log_level string
}

type ServerConfig struct {
	Logger       string
	Log_level    string
	Host_address string
}

type ProcessorConfig struct {
	Logger           string
	Next             []string
	Log_level        string
	Processor_type   string
	Address_list     []string
	IP_list          []string
	Command_list     []string
	File_path        string
	File_prefix      string
	File_name        string
	File_suffix      string
	File_compression string
	File_sizelimit   int64
	File_agelimit    int
	Redis_host       string
	Redis_password   string
	Redis_database   int64
	Zeromq_host      string
}

type ManagerConfig struct {
	Logger           string
	Repository       string
	Tracker          string
	Processor        []string
	Log_level        string
	Protocol_magic   uint32
	Protocol_version uint32
	Connection_rate  uint32
	Connection_limit uint32
}
