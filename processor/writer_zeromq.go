// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package processor

import (
	"sync"

	zmq "github.com/pebbe/zmq4"

	"github.com/CIRCL/pbtc/adaptor"
)

type ZeroMQWriter struct {
	Processor

	addr  string
	pub   *zmq.Socket
	lineQ chan string
	sig   chan struct{}
	wg    *sync.WaitGroup
}

func NewZeroMQWriter(options ...func(adaptor.Processor)) (*ZeroMQWriter, error) {
	w := &ZeroMQWriter{
		addr:  "127.0.0.1:12345",
		lineQ: make(chan string, 1),
		sig:   make(chan struct{}),
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

	return w, nil
}

func SetZeromqHost(addr string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*ZeroMQWriter)
		if !ok {
			return
		}

		w.addr = addr
	}
}

func (w *ZeroMQWriter) Start() {
	w.log.Info("[WZ] Start: begin")

	w.wg.Add(1)
	go w.goLines()

	w.log.Info("[WZ] Start: completed")
}

func (w *ZeroMQWriter) Stop() {
	w.log.Info("[WZ] Stop: begin")

	close(w.sig)
	w.wg.Wait()

	w.log.Info("[WZ] Stop: completed")
}

func (w *ZeroMQWriter) Process(record adaptor.Record) {
	w.lineQ <- record.String()
}

func (w *ZeroMQWriter) goLines() {
	defer w.wg.Done()

LineLoop:
	for {
		select {
		case _, ok := <-w.sig:
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
