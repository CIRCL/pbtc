package all

import (
	"net"
	"strconv"

	"github.com/btcsuite/btcd/wire"
)

type event interface {
}

type eventState struct {
	peer  *peer
	state uint32
}

type eventAddress struct {
	addr string
	list []string
}

func NewStateEvent(peer *peer, state uint32) *eventState {
	evt := &eventState{
		peer:  peer,
		state: state,
	}

	return evt
}

func NewAddressEvent(addr string, addrList []*wire.NetAddress) *eventAddress {
	list := make([]string, 0, len(addrList))
	for _, wAddr := range addrList {
		list = append(list, net.JoinHostPort(wAddr.IP.String(), strconv.Itoa(int(wAddr.Port))))
	}

	evt := &eventAddress{
		addr: addr,
		list: list,
	}

	return evt
}
