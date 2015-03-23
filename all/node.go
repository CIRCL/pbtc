package all

import (
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	pollInterval      = 60
	retryInitial      = 60 * time.Second
	retryMaximum      = 60 * time.Minute
	retryMultiplier   = 1.5
	retryRandomFactor = 0.5
)

const (
	stateIdle = iota
	stateConnecting
	stateOutWaitVer
	stateInWaitVer
	stateInWaitAck
	stateFailed
	stateConnected
)

type ConnectionRepository interface {
	Store(node Node)
	FindByIP(ip net.IP) Node
}

type Node struct {
	sendEx  chan wire.Message
	nodeOut chan string
	addr    string
	conn    net.Conn
	network wire.BitcoinNet
	version uint32
	nonce   uint64
	me      *wire.NetAddress
	you     *wire.NetAddress
	state   uint32
	backoff time.Duration
	timer   *time.Timer
	connMgr *ConnMgr
}

func NewNode(addr string, nodeOut chan string) (*Node, error) {
	node := &Node{
		addr:    addr,
		nodeOut: nodeOut,
		state:   stateIdle,
		backoff: retryInitial,
	}

	return node, nil
}

func (node *Node) Connect(connMgr *ConnMgr) {
	node.sendEx = make(chan wire.Message, 128)
	node.connMgr = connMgr

	conn, err := net.Dial("tcp", node.addr)
	if err != nil {
		log.Println("Failed to connect to", node.addr, err)
		go node.Retry()
		return
	}

	log.Println("Connected to", node.addr)
	node.backoff = retryInitial

	me, err := wire.NewNetAddress(conn.LocalAddr(), 0)
	if err != nil {
		log.Println("Failed to parse local address", node.addr, err)
		return
	}

	you, err := wire.NewNetAddress(conn.RemoteAddr(), 0)
	if err != nil {
		log.Println("Failed to parse remote address", node.addr, err)
		return
	}

	nonce, err := wire.RandomUint64()
	if err != nil {
		log.Println("Failed to generate random nonce", node.addr, err)
		return
	}

	node.conn = conn
	node.me = me
	node.you = you
	node.nonce = nonce

	go node.handleSend()
	go node.handleRecv()

	node.sendEx <- wire.NewMsgVersion(node.me, node.you, node.nonce, 0)
	node.state = stateOutWaitVer

	connMgr.AddNode(node)
}

func (node *Node) Retry() {
	node.timer = time.NewTimer(node.backoff)
	log.Println("Retrying", node.addr, "in", node.backoff)

	multiplier := time.Duration(retryMultiplier * float64(node.backoff.Seconds()))
	random := time.Duration(rand.Float64() * retryRandomFactor * float64(node.backoff.Seconds()))
	node.backoff = multiplier + random
	if node.backoff > retryMaximum {
		node.backoff = retryMaximum
	}

	_, ok := <-node.timer.C
	if !ok {
		node.backoff = retryInitial
		return
	}

	node.Connect(node.connMgr)
}

func (node *Node) Cancel() {
	node.timer.Stop()
}

func (node *Node) Disconnect() {
	node.conn.Close()

	close(node.sendEx)

	node.state = stateIdle
}

func (node *Node) Poll() {
	node.sendEx <- wire.NewMsgGetAddr()
}

func (node *Node) IsIdle() bool {
	if node.state == stateIdle {
		return true
	}

	return false
}

func (node *Node) IsConnected() bool {
	if node.state == stateConnected {
		return true
	}

	return false
}

func (node *Node) handleSend() {
	for msg := range node.sendEx {
		err := wire.WriteMessage(node.conn, msg, node.version, node.network)
		if err == io.EOF {
			log.Println("Connection closed remotely:", node.addr, err)
			node.Disconnect()
			break
		}
		if err != nil {
			log.Println("Could not send message:", node.addr, err)
			continue
		}

		log.Println("Message sent:", node.addr, msg.Command())
	}
}

func (node *Node) handleRecv() {
	for {
		msg, _, err := wire.ReadMessage(node.conn, node.version, node.network)
		if err == io.EOF {
			log.Println("Connection closed remotely:", node.addr, err)
			node.Disconnect()
			break
		}
		if err != nil {
			log.Println("Could not receive message:", node.addr, err)
			continue
		}

		log.Println("Message received:", node.addr, msg.Command())

		switch m := msg.(type) {
		case *wire.MsgVersion:
			if node.state == stateInWaitVer {
				if uint32(m.ProtocolVersion) < node.version {
					node.version = uint32(m.ProtocolVersion)
				}

				node.sendEx <- wire.NewMsgVersion(node.me, node.you, node.nonce, 0)
				node.state = stateInWaitAck
			} else if node.state == stateOutWaitVer {
				if uint32(m.ProtocolVersion) < node.version {
					node.version = uint32(m.ProtocolVersion)
				}

				node.sendEx <- wire.NewMsgVerAck()
				node.state = stateConnected
				node.sendEx <- wire.NewMsgAddr()
			}

		case *wire.MsgVerAck:
			if node.state == stateInWaitAck {
				node.state = stateConnected
				node.sendEx <- wire.NewMsgAddr()
			}

		case *wire.MsgPing:
			node.sendEx <- wire.NewMsgPong(m.Nonce)

		case *wire.MsgPong:

		case *wire.MsgGetAddr:
			node.sendEx <- wire.NewMsgAddr()

		case *wire.MsgAddr:
			for _, addr := range m.AddrList {
				node.nodeOut <- net.JoinHostPort(addr.IP.String(), strconv.Itoa(int(addr.Port)))
			}

		case *wire.MsgInv:

		case *wire.MsgGetHeaders:

		case *wire.MsgHeaders:

		case *wire.MsgGetBlocks:

		case *wire.MsgBlock:

		case *wire.MsgGetData:

		case *wire.MsgTx:

		case *wire.MsgAlert:

		default:
			log.Println("Unhandled message type:", msg.Command())
		}
	}
}
