package supervisor

import (
	"github.com/CIRCL/pbtc/adaptor"
)

type Supervisor struct {
	log    adaptor.Log
	loglib adaptor.Loglib
	repo   adaptor.Repository
	tkr    adaptor.Tracker
	svr    adaptor.Server
	mgr    adaptor.Manager
	pro    adaptor.Processor
}

func New(options ...func(*Supervisor)) (*Supervisor, error) {
	supervisor := &Supervisor{}

	for _, option := range options {
		option(supervisor)
	}

	supervisor.log = supervisor.loglib.GetLog("main")

	return supervisor, nil
}

func SetLogger(loglib adaptor.Loglib) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.loglib = loglib
	}
}

func SetRepository(repo adaptor.Repository) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.repo = repo
	}
}

func SetTracker(tkr adaptor.Tracker) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.tkr = tkr
	}
}

func SetServer(svr adaptor.Server) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.svr = svr
	}
}

func SetManager(mgr adaptor.Manager) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.mgr = mgr
	}
}

func SetProcessor(pro adaptor.Processor) func(*Supervisor) {
	return func(supervisor *Supervisor) {
		supervisor.pro = pro
	}
}

func (s *Supervisor) Close() {
}
