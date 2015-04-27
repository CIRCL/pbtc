package recorder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type OutputRecord struct {
	value   int64
	address btcutil.Address
}

func NewOutputRecord(txout *wire.TxOut) (*OutputRecord, error) {
	if txout == nil {
		return &OutputRecord{}, nil
	}

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(txout.PkScript,
		&chaincfg.TestNet3Params)
	if err != nil {
		return nil, err
	}
	if len(addrs) != 1 {
		return nil, errors.New("Multiple pub keys in txout not yet supported")
	}

	or := &OutputRecord{
		value:   txout.Value,
		address: addrs[0],
	}

	return or, nil
}

func (or *OutputRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(strconv.FormatInt(or.value, 10))
	buf.WriteString(or.address.EncodeAddress())

	return buf.String()
}

func (or *OutputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, or.value)
	binary.Write(buf, binary.LittleEndian,
		[]byte(or.address.EncodeAddress()))

	return buf.Bytes()
}
