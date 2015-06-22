package supervisor

import (
	"os"

	"code.google.com/p/gcfg"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/logger"
	"github.com/CIRCL/pbtc/manager"
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
		supervisor.repo[name] = initRepository(repo_cfg)
	}

	for name, tkr_cfg := range cfg.Tracker {
		supervisor.tkr[name] = initTracker(tkr_cfg)
	}

	for name, svr_cfg := range cfg.Server {
		supervisor.svr[name] = initServer(svr_cfg)
	}

	for name, pro_cfg := range cfg.Processor {
		supervisor.pro[name] = initProcessor(pro_cfg)
	}

	for name, mgr_cfg := range cfg.Manager {
		supervisor.mgr[name] = initManager(mgr_cfg)
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

	return supervisor, nil
}

func initLogger(lgr_cfg *LoggerConfig) (adaptor.Logger, error) {
	options := make([]func(*logger.GologgingLogger), 0, 2)

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
		file, err := os.Create(lgr_cfg.File_path)
		if err == nil {
			options = append(options, logger.SetFile(file))
		}

	}

	return logger.NewGologging(options...)
}

func initRepository(repo_cfg *RepositoryConfig) adaptor.Repository {
	return nil
}

func initTracker(tkr_cfg *TrackerConfig) adaptor.Tracker {
	return nil
}

func initServer(svr_cfg *ServerConfig) adaptor.Server {
	return nil
}

func initProcessor(pro_cfg *ProcessorConfig) adaptor.Processor {
	return nil
}

func initManager(mgr_cfg *ManagerConfig) adaptor.Manager {
	return nil
}

func (supervisor *Supervisor) Start() {
}

func (supervisor *Supervisor) Stop() {
}
