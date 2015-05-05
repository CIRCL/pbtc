package recorder

import (
	"bytes"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type OutputRecord struct {
	value int64
	addrs []btcutil.Address
}

func NewOutputRecord(txout *wire.TxOut) *OutputRecord {
	or := &OutputRecord{
		value: txout.Value,
		addrs: make([]btcutil.Address, 0),
	}

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(txout.PkScript,
		&chaincfg.MainNetParams)
	if err != nil {
		or.addrs = addrs
	}

	return or
}

func (or *OutputRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(strconv.FormatInt(or.value, 10))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(or.addrs)), 10))

	for _, addr := range or.addrs {
		buf.WriteString(" ")
		buf.WriteString(addr.EncodeAddress())
	}

	return buf.String()
}

func (or *OutputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)

	return buf.Bytes()
}
