package domain

import (
	"bytes"
	"encoding/gob"
	"net"
	"time"
)

type node struct {
	addr          *net.TCPAddr
	src           *net.TCPAddr
	numAttempts   uint32
	lastAttempted time.Time
	lastConnected time.Time
	lastSucceeded time.Time
}

// newNode creates a new node for the given address and source.
func newNode(addr *net.TCPAddr, src *net.TCPAddr) *node {
	n := &node{
		addr: addr,
		src:  src,
	}

	return n
}

func (node *node) String() string {
	return node.addr.String()
}

// GobEncode is required to implement the GobEncoder interface.
// It allows us to serialize the unexported fields of our nodes.
// We could also change them to exported, but as nodes are only
// handled internally in the repository, this is the better choice.
func (node *node) GobEncode() ([]byte, error) {
	buffer := &bytes.Buffer{}
	enc := gob.NewEncoder(buffer)

	err := enc.Encode(node.addr)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.src)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.numAttempts)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastAttempted)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastConnected)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(node.lastSucceeded)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// GobDecode is required to implement the GobDecoder interface.
// It allows us to deserialize the unexported fields of our nodes.
func (node *node) GobDecode(buf []byte) error {
	buffer := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&node.addr)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.src)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.numAttempts)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastAttempted)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastConnected)
	if err != nil {
		return err
	}

	err = dec.Decode(&node.lastSucceeded)
	if err != nil {
		return err
	}

	return nil
}
