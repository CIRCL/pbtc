package writer

import (
	"strconv"

	zmq "github.com/pebbe/zmq4"

	"github.com/CIRCL/pbtc/adaptor"
)

type ZeroMQWriter struct {
	port uint16
	pub  *zmq.Socket
	log  adaptor.Log
}

func NewZeroMQWriter(options ...func(*ZeroMQWriter)) (*ZeroMQWriter, error) {
	w := &ZeroMQWriter{
		port: 12345,
	}

	for _, option := range options {
		option(w)
	}

	pub, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return nil, err
	}

	addr := "tcp://*:" + strconv.FormatUint(uint64(w.port), 10)

	err = pub.Bind(addr)
	if err != nil {
		return nil, err
	}

	w.pub = pub

	return w, nil
}

func SetPort(port uint16) func(*ZeroMQWriter) {
	return func(w *ZeroMQWriter) {
		w.port = port
	}
}

func SetLogZMQ(log adaptor.Log) func(*ZeroMQWriter) {
	return func(w *ZeroMQWriter) {
		w.log = log
	}
}

func (w *ZeroMQWriter) Line(line string) {
	w.pub.Send(line, 0)
}
