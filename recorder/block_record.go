package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct{}

func NewBlockRecord(msg *wire.MsgBlock) *BlockRecord {
	/*header := msg.Header
	txlist := msg.Transactions*/

	return &BlockRecord{}
}

func (record *BlockRecord) String() string {
	return ""
}
