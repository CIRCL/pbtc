package domain

type ManagerStub struct {
	log Logger
}

func NewManagerStub(options ...func(*ManagerStub)) *ManagerStub {
	mgr := &ManagerStub{
		log: NewLoggerStub(),
	}

	return mgr
}

func (mgr *ManagerStub) Started(peer *Peer) {
	mgr.log.Debug("%v started", peer)
}

func (mgr *ManagerStub) Connected(peer *Peer) {
	mgr.log.Debug("%v connected", peer)
}

func (mgr *ManagerStub) Stopped(peer *Peer) {
	mgr.log.Debug("%v stopped", peer)
}
