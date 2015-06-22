package processor

import (
	"sync"
	"sync/atomic"

	zmq "github.com/pebbe/zmq4"

	"github.com/CIRCL/pbtc/adaptor"
)

type ZeroMQWriter struct {
	Processor

	addr  string
	pub   *zmq.Socket
	lineQ chan string
	wSig  chan struct{}
	wg    *sync.WaitGroup
	done  uint32
}

func NewZeroMQWriter(options ...func(adaptor.Processor)) (*ZeroMQWriter, error) {
	w := &ZeroMQWriter{
		addr:  "127.0.0.1:12345",
		lineQ: make(chan string, 1),
		wSig:  make(chan struct{}),
		wg:    &sync.WaitGroup{},
	}

	for _, option := range options {
		option(w)
	}

	pub, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return nil, err
	}

	addr := "tcp://" + w.addr

	err = pub.Bind(addr)
	if err != nil {
		return nil, err
	}

	w.pub = pub

	w.wg.Add(1)
	go w.goLines()

	return w, nil
}

func SetSocketAddress(addr string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*ZeroMQWriter)
		if !ok {
			return
		}

		w.addr = addr
	}
}

func (w *ZeroMQWriter) Close() {
	if atomic.SwapUint32(&w.done, 1) == 1 {
		return
	}

	close(w.wSig)

	w.wg.Wait()
}

func (w *ZeroMQWriter) Process(record adaptor.Record) {
	w.lineQ <- record.String()
}

func (w *ZeroMQWriter) goLines() {
	defer w.wg.Done()

LineLoop:
	for {
		select {
		case _, ok := <-w.wSig:
			if !ok {
				break LineLoop
			}

		case line := <-w.lineQ:
			_, err := w.pub.Send(line, 0)
			if err != nil {
				w.log.Error("Could not send line on zmq (%v)", err)
				continue
			}
		}
	}
}
