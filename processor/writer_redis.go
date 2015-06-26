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

	redis "gopkg.in/redis.v3"

	"github.com/CIRCL/pbtc/adaptor"
)

type RedisWriter struct {
	Processor

	lineQ  chan string
	wg     *sync.WaitGroup
	sig    chan struct{}
	client *redis.Client
	host   string
	pw     string
	db     int64
	done   uint32
}

func NewRedisWriter(options ...func(adaptor.Processor)) (*RedisWriter, error) {
	w := &RedisWriter{
		lineQ: make(chan string, 1),
		sig:   make(chan struct{}),
		wg:    &sync.WaitGroup{},
		host:  "127.0.0.1:23456",
		pw:    "",
		db:    0,
	}

	for _, option := range options {
		option(w)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     w.host,
		Password: w.pw,
		DB:       w.db,
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, err
	}

	w.client = client

	return w, nil
}

func SetRedisHost(host string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*RedisWriter)
		if !ok {
			return
		}

		w.host = host
	}
}

func SetRedisPassword(pw string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*RedisWriter)
		if !ok {
			return
		}

		w.pw = pw
	}
}

func SetRedisDatabase(db int64) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*RedisWriter)
		if !ok {
			return
		}

		w.db = db
	}
}

func (w *RedisWriter) Start() {
	w.wg.Add(1)
	go w.goProcess()
}

func (w *RedisWriter) Stop() {
	close(w.sig)
	w.wg.Wait()
}

func (w *RedisWriter) Process(record adaptor.Record) {
	w.lineQ <- record.String()
}

func (w *RedisWriter) goProcess() {
	defer w.wg.Done()

LineLoop:
	for {
		select {
		case _, ok := <-w.sig:
			if !ok {
				break LineLoop
			}

		case line := <-w.lineQ:
			err := w.client.Publish("", line).Err()
			if err != nil {
				w.log.Error("Could not send line to redis (%v)", err)
				continue
			}
		}
	}
}
