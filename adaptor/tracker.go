package adaptor

import (
	"github.com/btcsuite/btcd/wire"
)

type Tracker interface {
	Start()
	Stop()
	AddTx(hash wire.ShaHash)
	KnowsTx(hash wire.ShaHash) bool
	AddBlock(hash wire.ShaHash)
	KnowsBlock(hash wire.ShaHash) bool
}
