package peer

type Manager interface {
	Started(peer *Peer)
	Connected(peer *Peer)
	Stopped(peer *Peer)
}
