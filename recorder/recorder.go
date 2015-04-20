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
	case *wire.MsgAddr:
		rec.recordAddr(m)

	case *wire.MsgTx:
		rec.recordTx(m)

	case *wire.MsgHeaders:
		rec.recordHeaders(m)

	case *wire.MsgBlock:
		rec.recordBlock(m)

	}
}

func (rec *Recorder) recordAddr(msg *wire.MsgAddr) {

}

func (rec *Recorder) recordTx(msg *wire.MsgTx) {

}

func (rec *Recorder) recordHeaders(msg *wire.MsgHeaders) {

}

func (rec *Recorder) recordBlock(msg *wire.MsgBlock) {

}
