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

import (
	"errors"
	"time"

	"code.google.com/p/gcfg"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/manager"
	"github.com/CIRCL/pbtc/processor"
	"github.com/CIRCL/pbtc/repository"
	"github.com/CIRCL/pbtc/server"
	"github.com/CIRCL/pbtc/tracker"
)

type Supervisor struct {
	logr    map[string]adaptor.Logger
	repo    map[string]adaptor.Repository
	tkr     map[string]adaptor.Tracker
	svr     map[string]adaptor.Server
	pro     map[string]adaptor.Processor
	mgr     map[string]adaptor.Manager
	log     adaptor.Log
	options []interface{}
}

func New() (*Supervisor, error) {
	// load configuration file
	cfg := &Config{}
	err := gcfg.ReadFileInto(cfg, "pbtc.cfg")
	if err != nil {
		return nil, err
	}

	// initialize struct with maps
	supervisor := &Supervisor{
		logr: make(map[string]adaptor.Logger),
		repo: make(map[string]adaptor.Repository),
		tkr:  make(map[string]adaptor.Tracker),
		svr:  make(map[string]adaptor.Server),
		pro:  make(map[string]adaptor.Processor),
		mgr:  make(map[string]adaptor.Manager),
	}

	// initialize loggers so we can start logging
	for name, logr_cfg := range cfg.Logger {
		logr, err := initLogger(logr_cfg)
		if err != nil {
			continue
		}

		supervisor.logr[name] = logr
	}

	if len(supervisor.logr) == 0 {
		logr, err := logger.New()
		if err != nil {
			return nil, err
		}

		supervisor.logr[""] = logr
		supervisor.log = supervisor.logr[""].GetLog("sup")
		supervisor.log.Warning("No logger module defined")
	} else {
		_, ok := supervisor.logr[""]
		if !ok {
			for _, v := range supervisor.logr {
				supervisor.logr[""] = v
				supervisor.log = supervisor.logr[""].GetLog("sup")
				supervisor.log.Notice("No default logger defined")
				break
			}
		} else {
			supervisor.log = supervisor.logr[""].GetLog("sup")
		}
	}

	// initialize remaining modules
	for name, repo_cfg := range cfg.Repository {
		repo, err := initRepository(repo_cfg)
		if err != nil {
			continue
		}

		supervisor.repo[name] = repo
	}

	for name, tkr_cfg := range cfg.Tracker {
		tkr, err := initTracker(tkr_cfg)
		if err != nil {
			continue
		}

		supervisor.tkr[name] = tkr
	}

	for name, svr_cfg := range cfg.Server {
		svr, err := initServer(svr_cfg)
		if err != nil {
			continue
		}

		supervisor.svr[name] = svr
	}

	for name, pro_cfg := range cfg.Processor {
		pro, err := initProcessor(pro_cfg)
		if err != nil {
			continue
		}

		supervisor.pro[name] = pro
	}

	for name, mgr_cfg := range cfg.Manager {
		mgr, err := initManager(mgr_cfg)
		if err != nil {
			continue
		}

		supervisor.mgr[name] = mgr
	}

	// check remaining modules for missing values
	if len(supervisor.repo) == 0 {
		supervisor.log.Warning("No repository module defined")
		repo, err := repository.New()
		if err != nil {
			return nil, err
		}

		supervisor.repo[""] = repo
	}

	if len(supervisor.tkr) == 0 {
		supervisor.log.Warning("No tracker module defined")
		tkr, err := tracker.New()
		if err != nil {
			return nil, err
		}

		supervisor.tkr[""] = tkr
	}

	if len(supervisor.svr) == 0 {
		supervisor.log.Notice("No server module defined")
		svr, err := server.New()
		if err != nil {
			return nil, err
		}

		supervisor.svr[""] = svr
	}

	if len(supervisor.pro) == 0 {
		supervisor.log.Notice("No processor module defined")
	}

	if len(supervisor.mgr) == 0 {
		supervisor.log.Warning("No manager module defined")
		mgr, err := manager.New()
		if err != nil {
			return nil, err
		}

		supervisor.mgr[""] = mgr
	}

	// check remaining modules for missing default module
	_, ok := supervisor.repo[""]
	if !ok {
		for _, v := range supervisor.repo {
			supervisor.log.Notice("No default repository defined")
			supervisor.repo[""] = v
			break
		}
	}

	_, ok = supervisor.tkr[""]
	if !ok {
		for _, v := range supervisor.tkr {
			supervisor.log.Notice("No default tracker defined")
			supervisor.tkr[""] = v
			break
		}
	}

	_, ok = supervisor.svr[""]
	if !ok {
		for _, v := range supervisor.svr {
			supervisor.log.Notice("No default server defined")
			supervisor.svr[""] = v
			break
		}
	}

	_, ok = supervisor.mgr[""]
	if !ok {
		for _, v := range supervisor.mgr {
			supervisor.log.Notice("No default manager defined")
			supervisor.mgr[""] = v
			break
		}
	}

	// inject logging dependencies
	for key, logr := range supervisor.logr {
		logr_cfg, ok := cfg.Logger[key]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(logr_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "logr___" + key
		logr.SetLevel(log, level)
		logr.SetLog(logr.GetLog(log))
	}

	for key, repo := range supervisor.repo {
		repo_cfg, ok := cfg.Repository[key]
		if !ok {
			continue
		}

		logr, ok := supervisor.logr[repo_cfg.Logger]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(repo_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "repo___" + key
		logr.SetLevel(log, level)
		repo.SetLog(logr.GetLog(log))
	}

	for key, tkr := range supervisor.tkr {
		tkr_cfg, ok := cfg.Tracker[key]
		if !ok {
			continue
		}

		logr, ok := supervisor.logr[tkr_cfg.Logger]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(tkr_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "tkr___" + key
		logr.SetLevel(log, level)
		tkr.SetLog(logr.GetLog(log))
	}

	for key, svr := range supervisor.svr {
		svr_cfg, ok := cfg.Server[key]
		if !ok {
			continue
		}

		logr, ok := supervisor.logr[svr_cfg.Logger]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(svr_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "svr___" + key
		logr.SetLevel(log, level)
		svr.SetLog(logr.GetLog(log))
	}

	for key, pro := range supervisor.pro {
		pro_cfg, ok := cfg.Repository[key]
		if !ok {
			continue
		}

		logr, ok := supervisor.logr[pro_cfg.Logger]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(pro_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "pro___" + key
		logr.SetLevel(log, level)
		pro.SetLog(logr.GetLog(log))
	}

	for key, mgr := range supervisor.mgr {
		mgr_cfg, ok := cfg.Repository[key]
		if !ok {
			continue
		}

		logr, ok := supervisor.logr[mgr_cfg.Logger]
		if !ok {
			continue
		}

		level, err := logger.ParseLevel(mgr_cfg.Log_level)
		if err != nil {
			continue
		}

		log := "mgr___" + key
		logr.SetLevel(log, level)
		mgr.SetLog(logr.GetLog(log))
	}

	// inject processor dependencies

	return supervisor, nil
}

func initLogger(lgr_cfg *LoggerConfig) (adaptor.Logger, error) {
	options := make([]func(*logger.GologgingLogger), 0)

	if lgr_cfg.Console_enabled != false {
		enabled := lgr_cfg.Console_enabled
		options = append(options, logger.SetConsoleEnabled(enabled))
	}

	if lgr_cfg.Console_format != "" {
		format, err := logger.ParseFormat(lgr_cfg.Console_format)
		if err == nil {
			options = append(options, logger.SetConsoleFormat(format))
		}

	}

	if lgr_cfg.Console_level != "" {
		level, err := logger.ParseLevel(lgr_cfg.Console_level)
		if err == nil {
			options = append(options, logger.SetConsoleLevel(level))
		}
	}

	if lgr_cfg.File_enabled != false {
		enabled := lgr_cfg.File_enabled
		options = append(options, logger.SetFileEnabled(enabled))
	}

	if lgr_cfg.File_format != "" {
		format, err := logger.ParseFormat(lgr_cfg.File_format)
		if err == nil {
			options = append(options, logger.SetFileFormat(format))
		}
	}

	if lgr_cfg.File_level != "" {
		level, err := logger.ParseLevel(lgr_cfg.File_level)
		if err == nil {
			options = append(options, logger.SetFileLevel(level))
		}
	}

	if lgr_cfg.File_path != "" {
		path := lgr_cfg.File_path
		options = append(options, logger.SetFilePath(path))
	}

	return logger.NewGologging(options...)
}

func initRepository(repo_cfg *RepositoryConfig) (adaptor.Repository, error) {
	options := make([]func(*repository.Repository), 0)

	if repo_cfg.Seeds_list != nil {
		seeds := repo_cfg.Seeds_list
		options = append(options, repository.SetSeedsList(seeds...))
	}

	if repo_cfg.Seeds_port != 0 {
		port := repo_cfg.Seeds_port
		if port > 0 && port < 65535 {
			options = append(options, repository.SetSeedsPort(port))
		}
	}

	if repo_cfg.Backup_path != "" {
		path := repo_cfg.Backup_path
		options = append(options, repository.SetBackupPath(path))
	}

	if repo_cfg.Backup_rate != 0 {
		rate := time.Duration(repo_cfg.Backup_rate) * time.Second
		if rate > time.Minute*15 && rate < time.Hour*24 {
			options = append(options, repository.SetBackupRate(rate))
		}
	}

	if repo_cfg.Node_limit != 0 {
		limit := repo_cfg.Node_limit
		if limit > 1000 && limit < 1000000 {
			options = append(options, repository.SetNodeLimit(limit))
		}
	}

	return repository.New(options...)
}

func initTracker(tkr_cfg *TrackerConfig) (adaptor.Tracker, error) {
	options := make([]func(*tracker.Tracker), 0)

	return tracker.New(options...)
}

func initServer(svr_cfg *ServerConfig) (adaptor.Server, error) {
	options := make([]func(*server.Server), 0)

	if svr_cfg.Host_address != "" {
		host := svr_cfg.Host_address
		options = append(options, server.SetHostAddress(host))
	}

	return server.New(options...)
}

func initProcessor(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	pType, err := processor.ParseType(pro_cfg.Processor_type)
	if err != nil {
		return nil, err
	}

	switch pType {
	case processor.AddressF:
		return initAddressFilter(pro_cfg)

	case processor.CommandF:
		return initCommandFilter(pro_cfg)

	case processor.IPF:
		return initIPFilter(pro_cfg)

	case processor.FileW:
		return initFileWriter(pro_cfg)

	case processor.RedisW:
		return initRedisWriter(pro_cfg)

	case processor.ZeroMQW:
		return initZeroMQWriter(pro_cfg)

	default:
		return nil, errors.New("invalid processor type")
	}
}

func initAddressFilter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if len(pro_cfg.Address_list) > 0 {
		addresses := pro_cfg.Address_list
		options = append(options, processor.SetAddresses(addresses...))
	}

	return processor.NewAddressFilter(options...)
}

func initCommandFilter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if len(pro_cfg.Command_list) > 0 {
		commands := pro_cfg.Command_list
		options = append(options, processor.SetCommands(commands...))
	}

	return processor.NewCommandFilter(options...)
}

func initIPFilter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if len(pro_cfg.IP_list) > 0 {
		ips := pro_cfg.IP_list
		options = append(options, processor.SetIPs(ips...))
	}

	return processor.NewIPFilter(options...)
}

func initFileWriter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if pro_cfg.File_path != "" {
		path := pro_cfg.File_path
		options = append(options, processor.SetFilePath(path))
	}

	if pro_cfg.File_prefix != "" {
		prefix := pro_cfg.File_prefix
		options = append(options, processor.SetFilePrefix(prefix))
	}

	if pro_cfg.File_name != "" {
		name := pro_cfg.File_name
		options = append(options, processor.SetFileName(name))
	}

	if pro_cfg.File_suffix != "" {
		suffix := pro_cfg.File_suffix
		options = append(options, processor.SetFileSuffix(suffix))
	}

	if pro_cfg.File_sizelimit != 0 {
		sizelimit := pro_cfg.File_sizelimit
		options = append(options, processor.SetFileSizelimit(sizelimit))
	}

	if pro_cfg.File_agelimit != 0 {
		agelimit := time.Duration(pro_cfg.File_agelimit) * time.Second
		options = append(options, processor.SetFileAgelimit(agelimit))
	}

	return processor.NewFileWriter(options...)
}

func initRedisWriter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if pro_cfg.Redis_host != "" {
		host := pro_cfg.Redis_host
		options = append(options, processor.SetRedisHost(host))
	}

	if pro_cfg.Redis_password != "" {
		password := pro_cfg.Redis_password
		options = append(options, processor.SetRedisPassword(password))
	}

	if pro_cfg.Redis_database != 0 {
		database := pro_cfg.Redis_database
		options = append(options, processor.SetRedisDatabase(database))
	}

	return processor.NewRedisWriter(options...)
}

func initZeroMQWriter(pro_cfg *ProcessorConfig) (adaptor.Processor, error) {
	options := make([]func(adaptor.Processor), 0)

	if pro_cfg.Zeromq_host != "" {
		host := pro_cfg.Zeromq_host
		options = append(options, processor.SetZeromqHost(host))
	}

	return processor.NewZeroMQWriter(options...)
}

func initManager(mgr_cfg *ManagerConfig) (adaptor.Manager, error) {
	return nil, nil
}

func (supervisor *Supervisor) Start() {
	// start the module execution
	for _, logr := range supervisor.logr {
		logr.Start()
	}

	for _, repo := range supervisor.repo {
		repo.Start()
	}

	for _, tkr := range supervisor.tkr {
		tkr.Start()
	}

	for _, svr := range supervisor.svr {
		svr.Start()
	}

	for _, pro := range supervisor.pro {
		pro.Start()
	}

	for _, mgr := range supervisor.mgr {
		mgr.Start()
	}
}

func (supervisor *Supervisor) Stop() {
	// stop the module execution
	for _, mgr := range supervisor.mgr {
		mgr.Stop()
	}

	for _, pro := range supervisor.pro {
		pro.Stop()
	}

	for _, svr := range supervisor.svr {
		svr.Stop()
	}

	for _, tkr := range supervisor.tkr {
		tkr.Stop()
	}

	for _, repo := range supervisor.repo {
		repo.Stop()
	}

	for _, logr := range supervisor.logr {
		logr.Stop()
	}
}
