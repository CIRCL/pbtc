package supervisor

import (
	"code.google.com/p/gcfg"

	"github.com/CIRCL/pbtc/adaptor"
)

type Supervisor struct {
	log adaptor.Log
}

func New(logr adaptor.Logger) (*Supervisor, error) {
	supervisor := &Supervisor{
		log: logr.GetLog("supervisor"),
	}

	supervisor.log.Info("Loading configuration file")

	cfg := &Config{}
	err := gcfg.ReadFileInto(cfg, "pbtc.cfg")
	if err != nil {
		supervisor.log.Error("Could not load configuration file")
		return nil, err
	}

	if len(cfg.Logger) == 0 {
		supervisor.log.Warning("No logger module defined")
	}

	if len(cfg.Repository) == 0 {
		supervisor.log.Warning("No repository module defined")
	}

	if len(cfg.Tracker) == 0 {
		supervisor.log.Warning("No tracker module defined")
	}

	if len(cfg.Server) == 0 {
		supervisor.log.Notice("No server module defined")
	}

	if len(cfg.Manager) == 0 {
		supervisor.log.Warning("No manager module defined")
	}

	if len(cfg.Processor) == 0 {
		supervisor.log.Notice("No processor module defined")
	}

	return supervisor, nil
}

func (supervisor *Supervisor) Start() {
}

func (supervisor *Supervisor) Stop() {
}
