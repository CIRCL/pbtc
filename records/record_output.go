package records

import (
	"bytes"
	"strconv"

	"github.com/btcsuite/btcutil"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type OutputRecord struct {
	value int64
	class uint8
	sigs  uint8
	addrs []btcutil.Address
}

func NewOutputRecord(txout *wire.TxOut) *OutputRecord {
	class, addrs, sigs, _ := txscript.ExtractPkScriptAddrs(txout.PkScript,
		&chaincfg.MainNetParams)

	record := &OutputRecord{
		value: txout.Value,
		class: uint8(class),
		sigs:  uint8(sigs),
		addrs: addrs,
	}

	return record
}

func (or *OutputRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatInt(or.value, 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(ParseClass(or.class))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatUint(uint64(or.sigs), 10))
	buf.WriteString(Delimiter3)
	buf.WriteString(strconv.FormatInt(int64(len(or.addrs)), 10))
	for _, addr := range or.addrs {
		buf.WriteString(Delimiter3)
		buf.WriteString(addr.EncodeAddress())
	}

	return buf.String()
}
