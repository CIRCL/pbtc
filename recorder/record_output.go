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
	buf.WriteString(" ")
	buf.WriteString(ParseClass(or.class))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(uint64(or.sigs), 10))
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
	binary.Write(buf, binary.LittleEndian, or.value)
	binary.Write(buf, binary.LittleEndian, len(or.addrs))

	for _, addr := range or.addrs {
		binary.Write(buf, binary.LittleEndian, addr.ScriptAddress())
	}

	return buf.Bytes()
}
