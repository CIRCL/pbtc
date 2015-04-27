package recorder

import (
	"github.com/btcsuite/btcd/wire"
)

type TransactionRecord struct {
}

func NewTransactionRecord(msg *wire.MsgTx) *TransactionRecord {
	/*version := msg.Version
	txinlist := msg.TxIn
	txoutlist := msg.TxOut
	locktime := msg.LockTime*/

	return &TransactionRecord{}
}

func (record *TransactionRecord) String() string {
	return ""
}
