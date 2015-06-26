package processor

import (
	"sync"
	"sync/atomic"

	redis "gopkg.in/redis.v3"

	"github.com/CIRCL/pbtc/adaptor"
)

type RedisWriter struct {
	Processor

	lineQ  chan string
	wg     *sync.WaitGroup
	wSig   chan struct{}
	client *redis.Client
	host   string
	pw     string
	db     int64
	done   uint32
}

func NewRedisWriter(options ...func(adaptor.Processor)) (*RedisWriter, error) {
	w := &RedisWriter{
		lineQ: make(chan string, 1),
		wSig:  make(chan struct{}),
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

	w.wg.Add(1)
	go w.goLines()

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

func (w *RedisWriter) Close() {
	if atomic.SwapUint32(&w.done, 1) == 1 {
		return
	}

	close(w.wSig)

	w.wg.Wait()
}

func (w *RedisWriter) Process(record adaptor.Record) {
	w.lineQ <- record.String()
}

func (w *RedisWriter) goLines() {
	defer w.wg.Done()

LineLoop:
	for {
		select {
		case _, ok := <-w.wSig:
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
