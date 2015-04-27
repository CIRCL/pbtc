package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type AddressRecord struct{}

func NewAddressRecord(addr *wire.NetAddress) *AddressRecord {
	return &AddressRecord{}
}

func (record *AddressRecord) String() string {
	return ""
}
