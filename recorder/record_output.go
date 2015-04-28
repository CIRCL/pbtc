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
	value     int64
	addr_list []btcutil.Address
	success   bool
}

func NewOutputRecord(txout *wire.TxOut) *OutputRecord {
	or := &OutputRecord{
		value:   txout.Value,
		success: true,
	}

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(txout.PkScript,
		&chaincfg.MainNetParams)
	if err != nil {
		or.success = false
		return or
	}

	or.addr_list = addrs
	return or
}

func (or *OutputRecord) String() string {
	buf := new(bytes.Buffer)

	if !or.success {
		buf.WriteString("0")
	} else {
		buf.WriteString(strconv.FormatUint(uint64(len(or.addr_list)), 10))

		for _, addr := range or.addr_list {
			buf.WriteString(" ")
			buf.WriteString(addr.EncodeAddress())
		}
	}

	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(or.value, 10))

	return buf.String()
}

func (or *OutputRecord) Bytes() []byte {
	buf := new(bytes.Buffer)

	if !or.success {
		binary.Write(buf, binary.LittleEndian, 0)
	} else {
		binary.Write(buf, binary.LittleEndian, len(or.addr_list))

		for _, addr := range or.addr_list {
			binary.Write(buf, binary.LittleEndian, addr.ScriptAddress())
		}
	}

	binary.Write(buf, binary.LittleEndian, or.value)

	return buf.Bytes()
}
