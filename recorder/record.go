package recorder

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type Record interface {
	String() string
	Bytes() []byte
}

const (
	MsgAddr = iota
	MsgAlert
	MsgBlock
	MsgHeaders
	MsgInv
	MsgPing
	MsgPong
	MsgReject
	MsgVersion
	MsgTx
	MsgFilterAdd
	MsgFilterLoad
	MsgFilterClear
	MsgGetAddr
	MsgGetBlocks
	MsgGetData
	MsgGetHeaders
	MsgMemPool
	MsgMerkleBlock
	MsgNotFound
	MsgVerAck
	MsgUnknown
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

func ParseCommand(command string) uint8 {
	switch command {
	case wire.CmdAddr:
		return MsgAddr

	case wire.CmdAlert:
		return MsgAlert

	case wire.CmdBlock:
		return MsgBlock

	case wire.CmdHeaders:
		return MsgHeaders

	case wire.CmdInv:
		return MsgInv

	case wire.CmdPing:
		return MsgPing

	case wire.CmdPong:
		return MsgPong

	case wire.CmdReject:
		return MsgReject

	case wire.CmdVersion:
		return MsgVersion

	case wire.CmdTx:
		return MsgTx

	case wire.CmdFilterAdd:
		return MsgFilterAdd

	case wire.CmdFilterLoad:
		return MsgFilterLoad

	case wire.CmdFilterClear:
		return MsgFilterClear

	case wire.CmdGetAddr:
		return MsgGetAddr

	case wire.CmdGetBlocks:
		return MsgGetBlocks

	case wire.CmdGetData:
		return MsgGetData

	case wire.CmdGetHeaders:
		return MsgGetHeaders

	case wire.CmdMemPool:
		return MsgMemPool

	case wire.CmdMerkleBlock:
		return MsgMerkleBlock

	case wire.CmdNotFound:
		return MsgNotFound

	case wire.CmdVerAck:
		return MsgVerAck
	}

	return MsgUnknown
}
