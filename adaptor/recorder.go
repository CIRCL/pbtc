package adaptor

import (
	"github.com/btcsuite/btcd/wire"
)

type Recorder interface {
	Message(msg wire.Message)
}
