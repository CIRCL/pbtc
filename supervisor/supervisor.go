package supervisor

import (
	"errors"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/repository"
	"github.com/CIRCL/pbtc/tracker"
)

type Supervisor struct {
	log  adaptor.Log
	repo map[string]adaptor.Repository
	trk  map[string]adaptor.Tracker
	svr  map[string]adaptor.Server
	mgr  map[string]adaptor.Manager
	pro  map[string]adaptor.Processor
}

func New(cfg Config) (*Supervisor, error) {
	// read the configuration file from disk
	cfg := &Config{}
	err := gcfg.ReadFileInto(cfg, "pbtc.cfg")
	if err != nil {
		return nil, err
	}

	// check if we have at least one definition for each module
	if len(cfg.Repository) == 0 {
		return nil, errors.New("No repository module defined")
	}

	if len(cfg.Tracker) == 0 {
		return nil, errors.New("No tracker module defined")
	}

	if len(cfg.Server) == 0 {
		return nil, errors.New("No server module defined")
	}

	if len(cfg.Manager) == 0 {
		return nil, errors.New("No manager module defined")
	}

	if len(cfg.Processor) == 0 {
		return nil, errors.New("No processor module defined")
	}

	if cfg.Repository[""] == nil {
		for k, v := range cfg.Repository {
			cfg.Repository[""] = v
			delete(cfg.Repository, k)
			break
		}
	}

	if cfg.Tracker[""] == nil {
		for k, v := range cfg.Tracker {
			cfg.Tracker[""] = v
			delete(cfg.Tracker, k)
			break
		}
	}

	if cfg.Server[""] == nil {
		for k, v := range cfg.Server {
			cfg.Server[""] = v
			delete(cfg.Server, k)
			break
		}
	}

	if cfg.Manager[""] == nil {
		for k, v := range cfg.Manager {
			cfg.Manager[""] = v
			delete(cfg.Manager, k)
			break
		}
	}

	if cfg.Processor[""] == nil {
		for k, v := range cfg.Processor {
			cfg.Processor[""] = v
			delete(cfg.Processor, k)
			break
		}
	}

	// initialize supervisor struct
	sup := &Supervisor{
		log:  logr.GetLog("sup"),
		repo: make(map[string]adaptor.Repository),
		trk:  make(map[string]adaptor.Tracker),
		svr:  make(map[string]adaptor.Server),
		mgr:  make(map[string]adaptor.Manager),
		pro:  make(map[string]adaptor.Processor),
	}

	// initialize repositories
	for k, v := range cfg.Repository {
		repo, err := repository.New(
			repository.SetLog(logr.GetLog("repo"+k)),
			repository.SetSeedsList(v.Seeds_list...),
			repository.SetSeedsPort(v.Seeds_port),
			repository.SetBackupPath(v.Backup_path),
			repository.SetBackupRate(v.Backup_rate),
			repository.SetNodeLimit(v.Node_limit),
		)
		if err != nil {
			return nil, err
		}

		logr.SetLevel("repo"+k, v.Log_level)
		sup.repo[k] = repo
	}

	// initialize trackers
	for k, v := range cfg.Tracker {
		tkr, err := tracker.New(
			tracker.SetLog(logr.GetLog("tkr" + k)),
		)
		if err != nil {
			return nil, err
		}

		logr.SetLevel("tkr"+k, tkr.Log_level)
	}

	// initialize servers

	// initialize managers

	// initialize processors

	return sup, nil
}

func (supervisor *Supervisor) Start() {
}

func (supervisor *Supervisor) Stop() {
}
