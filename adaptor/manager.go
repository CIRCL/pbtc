package adaptor

type Manager interface {
	Connected(p Peer)
	Ready(p Peer)
	Stopped(p Peer)
}
