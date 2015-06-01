package convertor

import (
	"net"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/records"
)

func Message(msg wire.Message, r *net.TCPAddr, l *net.TCPAddr) adaptor.Record {
	switch m := msg.(type) {
	case *wire.MsgAddr:
		return records.NewAddressRecord(m, r, l)

	case *wire.MsgAlert:
		return records.NewAlertRecord(m, r, l)

	case *wire.MsgBlock:
		return records.NewBlockRecord(m, r, l)

	case *wire.MsgHeaders:
		return records.NewHeadersRecord(m, r, l)

	case *wire.MsgInv:
		return records.NewInventoryRecord(m, r, l)

	case *wire.MsgPing:
		return records.NewPingRecord(m, r, l)

	case *wire.MsgPong:
		return records.NewPongRecord(m, r, l)

	case *wire.MsgReject:
		return records.NewRejectRecord(m, r, l)

	case *wire.MsgVersion:
		return records.NewVersionRecord(m, r, l)

	case *wire.MsgTx:
		return records.NewTransactionRecord(m, r, l)

	case *wire.MsgFilterAdd:
		return records.NewFilterAddRecord(m, r, l)

	case *wire.MsgFilterClear:
		return records.NewFilterClearRecord(m, r, l)

	case *wire.MsgFilterLoad:
		return records.NewFilterLoadRecord(m, r, l)

	case *wire.MsgGetAddr:
		return records.NewGetAddrRecord(m, r, l)

	case *wire.MsgGetBlocks:
		return records.NewGetBlocksRecord(m, r, l)

	case *wire.MsgGetData:
		return records.NewGetDataRecord(m, r, l)

	case *wire.MsgGetHeaders:
		return records.NewGetHeadersRecord(m, r, l)

	case *wire.MsgMemPool:
		return records.NewMemPoolRecord(m, r, l)

	case *wire.MsgMerkleBlock:
		return records.NewMerkleBlockRecord(m, r, l)

	case *wire.MsgNotFound:
		return records.NewNotFoundRecord(m, r, l)

	case *wire.MsgVerAck:
		return records.NewVerAckRecord(m, r, l)

	default:
		return nil
	}
}
