package domain

import (
	"log"
	"net"
	"time"

	"github.com/CIRCL/pbtc/usecases"
)

type Initializer struct {
	nodeIn  <-chan string
	connIn  <-chan net.Conn
	peerOut chan<- *usecases.Peer
}

func NewInitializer(nodeIn <-chan string, connIn <-chan net.Conn, peerOut chan<- *usecases.Peer) (*Initializer, error) {
	init := &Initializer{
		nodeIn:  nodeIn,
		connIn:  connIn,
		peerOut: peerOut,
	}

	return init, nil
}

func (init *Initializer) Start() {
	go func() {
		for {
			select {
			case node, ok := <-init.nodeIn:
				if !ok {
					init.nodeIn = nil
					continue
				}

				init.addOutgoing(node)

			case conn, ok := <-init.connIn:
				if !ok {
					init.connIn = nil
					continue
				}

				init.addIncoming(conn)
			}

			if init.nodeIn == nil && init.connIn == nil {
				break
			}
		}

		close(init.peerOut)
	}()
}

func (init *Initializer) addOutgoing(node string) {
	conn, err := net.DialTimeout("tcp", node+":8333", time.Second)
	if err != nil {
		log.Println(err)
		return
	}

	_ = conn

	// send our version

	// wait for ack

	// wait for peer version

	// send ack

	peer := &usecases.Peer{}
	init.peerOut <- peer
}

func (init *Initializer) addIncoming(conn net.Conn) {

	// wait for peer version

	// send ack

	// send our version

	// wait for ack

	peer := &usecases.Peer{}
	init.peerOut <- peer
}

/*if outbound {
	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		log.Println(err)
		return
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		log.Println(err)
		return
	}

	msg := wire.NewMsgVersion(me, you, server.nonce, 0)
	wire.WriteMessage(conn, msg, server.version, server.network)
}

for atomic.LoadUint32(&server.shutdown) == 0 {
	msg, _, err := wire.ReadMessage(conn, server.version, server.network)
	if err == io.EOF {
		log.Println(err)
		break
	}

	if err != nil {
		log.Println(err)
		continue
	}

	log.Println(msg)
}*/
