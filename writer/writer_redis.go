package writer

import (
	"sync"
	"sync/atomic"

	redis "gopkg.in/redis.v3"

	"github.com/CIRCL/pbtc/adaptor"
)

type RedisWriter struct {
	log    adaptor.Log
	lineQ  chan string
	wg     *sync.WaitGroup
	wSig   chan struct{}
	client *redis.Client
	addr   string
	pw     string
	db     int64
	done   uint32
}

func NewRedis(options ...func(*RedisWriter)) (*RedisWriter, error) {
	w := &RedisWriter{
		lineQ: make(chan string, 1),
		wSig:  make(chan struct{}),
		wg:    &sync.WaitGroup{},
		addr:  "127.0.0.1:23456",
		pw:    "",
		db:    0,
	}

	for _, option := range options {
		option(w)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     w.addr,
		Password: w.pw,
		DB:       w.db,
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, err
	}

	w.client = client

	w.startup()

	return w, nil
}

func SetLogRedis(log adaptor.Log) func(*RedisWriter) {
	return func(w *RedisWriter) {
		w.log = log
	}
}

func SetAddressRedis(addr string) func(*RedisWriter) {
	return func(w *RedisWriter) {
		w.addr = addr
	}
}

func SetPassword(pw string) func(*RedisWriter) {
	return func(w *RedisWriter) {
		w.pw = pw
	}
}

func SetDatabase(db int64) func(*RedisWriter) {
	return func(w *RedisWriter) {
		w.db = db
	}
}

func (w *RedisWriter) Stop() {
	if atomic.SwapUint32(&w.done, 1) == 1 {
		return
	}

	close(w.wSig)

	w.wg.Wait()
}

func (w *RedisWriter) Line(line string) {
	w.lineQ <- line
}

func (w *RedisWriter) startup() {
	w.wg.Add(1)

	go w.goLines()
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
