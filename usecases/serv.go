package usecases

import (
	"errors"
	"log"
	"net"

	"github.com/btcsuite/btcd/wire"
)

type Server struct {
	listIn    chan string
	peerIn    chan *Peer
	listeners []net.Listener
	peers     []*Peer
	Network   wire.BitcoinNet
	Version   uint32
	nonce     uint64
}

func NewServer(network wire.BitcoinNet, version uint32) (*Server, error) {
	nonce, err := wire.RandomUint64()
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not initialize server nonce")
	}

	var listeners []net.Listener
	var peers []*Peer

	serv := &Server{
		Network:   network,
		Version:   version,
		listeners: listeners,
		peers:     peers,
		nonce:     nonce,
	}

	return serv, nil
}

func (serv *Server) Start(listIn chan string, peerIn chan *Peer) {
	serv.listIn = listIn
	serv.peerIn = peerIn

	go serv.handleListeners()
	go serv.handlePeers()
}

func (serv *Server) Stop() {
	for _, listener := range serv.listeners {
		listener.Close()
	}

	for _, peer := range serv.peers {
		peer.Stop()
	}

	close(serv.listIn)
	close(serv.peerIn)
}

func (serv *Server) handleListeners() {
	go func() {
		for list := range serv.listIn {
			listener, err := net.Listen("tcp4", list+":18333")
			if err != nil {
				log.Println(err)
				continue
			}

			serv.listeners = append(serv.listeners, listener)
			log.Println("Now listening on IPs:", len(serv.listeners))
		}
	}()
}

func (serv *Server) handlePeers() {
	go func() {
		for peer := range serv.peerIn {
			peer.Start()

			serv.peers = append(serv.peers, peer)
			log.Println("Now talking to peers:", len(serv.peers))
		}
	}()
}

func (serv *Server) AddListener(list string) {
	serv.listIn <- list
}

func (serv *Server) AddPeer(peer *Peer) {
	serv.peerIn <- peer
}
