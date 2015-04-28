package recorder

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type OutputRecord struct {
	value   int64
	address btcutil.Address
	code    uint32
}

func NewOutputRecord(txout *wire.TxOut) *OutputRecord {
	or := &OutputRecord{
		value: txout.Value,
	}

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(txout.PkScript,
		&chaincfg.MainNetParams)
	if err != nil {
		or.code = 1
	}
	if len(addrs) != 1 {
		or.code = 2
	}

	if or.code == 0 {
		or.address = addrs[0]
	}

	return or
}

func (or *OutputRecord) String() string {
	buf := new(bytes.Buffer)

	switch or.code {
	case 0:
		buf.WriteString(or.address.EncodeAddress())

	case 1:
		buf.WriteString("complexescript....................")

	case 2:
		buf.WriteString("multisig..........................")

	default:
		buf.WriteString("extractfailure....................")
	}

	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(or.value, 10))

	return buf.String()
}

func (or *OutputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, or.code)

	if or.code == 0 {
		binary.Write(buf, binary.LittleEndian, or.address.ScriptAddress())
	}

	binary.Write(buf, binary.LittleEndian, or.value)

	return buf.Bytes()
}
