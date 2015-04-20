package adaptor

type Manager interface {
	Started(p Peer)
	Ready(p Peer)
	Stopped(p Peer)
}
