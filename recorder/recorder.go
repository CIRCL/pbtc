package recorder

import (
	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
)

type Recorder struct {
	cmdConfig map[string]bool

	log adaptor.Logger
}

func New(options ...func(*Recorder)) (*Recorder, error) {
	rec := &Recorder{
		cmdConfig: make(map[string]bool),
	}

	for _, option := range options {
		option(rec)
	}

	return rec, nil
}

func SetLogger(log adaptor.Logger) func(*Recorder) {
	return func(rec *Recorder) {
		rec.log = log
	}
}

func SetTypes(cmds ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

func (rec *Recorder) Message(msg wire.Message) {
	if !rec.cmdConfig[msg.Command()] {
		return
	}

	switch m := msg.(type) {
	case *wire.MsgVersion:
		rec.recordVersion(m)

	case *wire.MsgAddr:
		rec.recordAddr(m)

	case *wire.MsgInv:
		rec.recordInv(m)

	case *wire.MsgTx:
		rec.recordTx(m)
	}
}

func (rec *Recorder) recordVersion(msg *wire.MsgVersion) {
	record := NewVersionRecord(msg)
	rec.log.Debug(record.String())
}

func (rec *Recorder) recordVerAck(msg *wire.MsgVerAck) {

}

func (rec *Recorder) recordAddr(msg *wire.MsgAddr) {
	record := NewAddressRecord(msg)
	rec.log.Debug(record.String())
}

func (rec *Recorder) recordInv(msg *wire.MsgInv) {
	record := NewInventoryRecord(msg)
	rec.log.Debug(record.String())
}

func (rec *Recorder) recordGetData(msg *wire.MsgGetData) {

}

func (rec *Recorder) recordNotFound(msg *wire.MsgNotFound) {

}

func (rec *Recorder) recordGetBlocks(msg *wire.MsgGetBlocks) {

}

func (rec *Recorder) recordGetHeaders(msg *wire.MsgGetHeaders) {

}

func (rec *Recorder) recordTx(msg *wire.MsgTx) {
	record := NewTransactionRecord(msg)
	rec.log.Debug(record.String())
}

func (rec *Recorder) recordBlock(msg *wire.MsgBlock) {

}

func (rec *Recorder) recordHeaders(msg *wire.MsgHeaders) {

}

func (rec *Recorder) recordGetAddr(msg *wire.MsgHeaders) {

}

func (rec *Recorder) recordMemPool(msg *wire.MsgMemPool) {

}

func (rec *Recorder) recordPing(msg *wire.MsgPing) {

}

func (rec *Recorder) recordPong(msg *wire.MsgPong) {

}

func (rec *Recorder) recordReject(msg *wire.MsgReject) {

}

func (rec *Recorder) recordFilterLoad(msg *wire.MsgFilterLoad) {

}

func (rec *Recorder) recordFilterAdd(msg *wire.MsgFilterAdd) {

}

func (rec *Recorder) recordFilterClear(msg *wire.MsgFilterClear) {

}

func (rec *Recorder) recordMerkleBlock(msg *wire.MsgMerkleBlock) {

}

func (rec *Recorder) recordAlert(msg *wire.MsgAlert) {

}
