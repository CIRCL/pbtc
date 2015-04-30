package parmap

import (
	"fmt"
	"hash/fnv"
	"sync"
)

type ParMap struct {
	count  uint32
	shards []*Shard
}

type Shard struct {
	index map[string]fmt.Stringer
	mutex *sync.RWMutex
}

func New(options ...func(*ParMap)) *ParMap {
	pm := &ParMap{
		count: 32,
	}

	for _, option := range options {
		option(pm)
	}

	pm.shards = make([]*Shard, pm.count)

	for i := 0; i < int(pm.count); i++ {
		pm.shards[i] = newShard()
	}

	return pm
}

func newShard() *Shard {
	shard := &Shard{
		index: make(map[string]fmt.Stringer),
		mutex: &sync.RWMutex{},
	}

	return shard
}

func SetShardCount(count uint32) func(*ParMap) {
	return func(pm *ParMap) {
		pm.count = count
	}
}

func (pm *ParMap) getShard(key string) *Shard {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	shard := pm.shards[hasher.Sum32()%pm.count]

	return shard
}

func (pm *ParMap) Insert(item fmt.Stringer) {
	key := item.String()
	shard := pm.getShard(key)
	shard.mutex.Lock()
	shard.index[key] = item
	shard.mutex.Unlock()
}

func (pm *ParMap) Get(key string) (fmt.Stringer, bool) {
	shard := pm.getShard(key)
	shard.mutex.RLock()
	item, ok := shard.index[key]
	shard.mutex.RUnlock()

	return item, ok
}

func (pm *ParMap) Count() int {
	count := 0
	for _, shard := range pm.shards {
		shard.mutex.RLock()
		count += len(shard.index)
		shard.mutex.RUnlock()
	}

	return count
}

func (pm *ParMap) Has(item fmt.Stringer) bool {
	key := item.String()
	return pm.HasKey(key)
}

func (pm *ParMap) HasKey(key string) bool {
	shard := pm.getShard(key)
	shard.mutex.RLock()
	_, ok := shard.index[key]
	shard.mutex.RUnlock()

	return ok
}

func (pm *ParMap) Remove(item fmt.Stringer) {
	key := item.String()
	pm.RemoveKey(key)
}

func (pm *ParMap) RemoveKey(key string) {
	shard := pm.getShard(key)
	shard.mutex.Lock()
	delete(shard.index, key)
	shard.mutex.Unlock()
}

func (pm *ParMap) Iter() <-chan fmt.Stringer {
	c := make(chan fmt.Stringer)

	go func() {
		for _, shard := range pm.shards {
			shard.mutex.RLock()
			for _, item := range shard.index {
				c <- item
			}
			shard.mutex.RUnlock()
		}
		close(c)
	}()

	return c
}
