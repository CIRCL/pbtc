package usecases

import (
	"errors"
	"log"
	"net"

	"github.com/btcsuite/btcd/wire"
)

type Server struct {
	listIn    <-chan string
	peerIn    <-chan *Peer
	listeners []net.Listener
	peers     []*Peer
	version   uint32
	network   wire.BitcoinNet
	nonce     uint64
}

func NewServer(listIn <-chan string, peerIn <-chan *Peer) (*Server, error) {
	nonce, err := wire.RandomUint64()
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not initialize server nonce")
	}

	var listeners []net.Listener
	var peers []*Peer

	server := &Server{
		listIn:    listIn,
		peerIn:    peerIn,
		listeners: listeners,
		peers:     peers,
		version:   wire.ProtocolVersion,
		network:   wire.TestNet3,
		nonce:     nonce,
	}

	return server, nil
}

func (server *Server) Start() {
	go func() {
		for {
			select {
			case peer, ok := <-server.peerIn:
				if !ok {
					server.peerIn = nil
					continue
				}

				server.addPeer(peer)

			case list, ok := <-server.listIn:
				if !ok {
					server.listIn = nil
					continue
				}

				server.addListener(list)
			}

			if server.peerIn == nil && server.listIn == nil {
				for _, peer := range server.peers {
					peer.Close()
				}

				for _, listener := range server.listeners {
					listener.Close()
				}

				break
			}
		}
	}()
}

func (server *Server) addListener(list string) {
	listener, err := net.Listen("tcp4", list+":8333")
	if err != nil {
		log.Println(err)
		return
	}

	server.listeners = append(server.listeners, listener)
	log.Println("Listening on ", len(server.listeners), " addresses")
}

func (server *Server) addPeer(peer *Peer) {
	server.peers = append(server.peers, peer)
	log.Println("Connected to ", len(server.peers), " peers")
}
