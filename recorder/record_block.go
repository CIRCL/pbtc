package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
}

func NewBlockRecord(msg *wire.MsgBlock) *BlockRecord {
	br := &BlockRecord{}

	return br
}

func (br *BlockRecord) String() string {
	return ""
}

func (br *BlockRecord) Bytes() []byte {
	return nil
}
