package domain

import (
	"errors"
	"log"
	"net"
)

type Server struct {
	listeners []net.Listener
}

func NewServer(ips []net.IP) (*Server, error) {
	var listeners []net.Listener

	if len(ips) == 0 {
		return nil, errors.New("No server IPs provided")
	}

	for _, ip := range ips {
		listener, err := net.Listen("tcp4", ip.String()+":8333")
		if err != nil {
			log.Println(err)
			continue
		}

		listeners = append(listeners, listener)
	}

	if len(listeners) == 0 {
		return nil, errors.New("Could not listen on any provided IP")
	}

	server := &Server{
		listeners: listeners,
	}

	return server, nil
}
