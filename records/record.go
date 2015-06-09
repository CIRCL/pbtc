package records

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/txscript"
)

const (
	Version = "PBTC LOG VERSION 1.0"
)

const (
	Delimiter1 = "|"
	Delimiter2 = ","
	Delimiter3 = "|"
)

func ParseClass(class uint8) string {
	newclass := txscript.ScriptClass(class)

	switch newclass {
	case txscript.NonStandardTy:
		return "nonstandard"

	case txscript.PubKeyTy:
		return "pubkey"

	case txscript.PubKeyHashTy:
		return "pubkeyhash"

	case txscript.ScriptHashTy:
		return "scripthash"

	case txscript.MultiSigTy:
		return "multisig"

	case txscript.NullDataTy:
		return "nulldata"

	default:
		return "invalid"
	}
}

type Record struct {
	stamp time.Time
	la    *net.TCPAddr
	ra    *net.TCPAddr
	cmd   string
}

func (r *Record) Timestamp() time.Time {
	return r.stamp
}

func (r *Record) RemoteAddress() *net.TCPAddr {
	return r.ra
}

func (r *Record) LocalAddress() *net.TCPAddr {
	return r.la
}

func (r *Record) Command() string {
	return r.cmd
}
