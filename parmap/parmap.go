package parmap

import (
	"fmt"
	"hash/fnv"
	"sync"
)

// ParMap implements a sharded & synchronized hash map for items that have a
// string representation. It uses the string representation as key.
type ParMap struct {
	count  uint32
	shards []*shard
}

// New creates a new sharded & synchronized hash map with the given options.
func New(options ...func(*ParMap)) *ParMap {
	pm := &ParMap{
		count: 32,
	}

	for _, option := range options {
		option(pm)
	}

	pm.shards = make([]*shard, pm.count)

	for i := 0; i < int(pm.count); i++ {
		pm.shards[i] = newShard()
	}

	return pm
}

// SetShardCount sets the number of separately synchronized shards.
func SetShardCount(count uint32) func(*ParMap) {
	return func(pm *ParMap) {
		pm.count = count
	}
}

// Insert adds a new item to the hash map or, if the key already exists,
// replaces the current item with the new one.
func (pm *ParMap) Insert(item fmt.Stringer) {
	key := item.String()
	shard := pm.getShard(key)
	shard.mutex.Lock()
	shard.index[key] = item
	shard.mutex.Unlock()
}

// Get returns the item with the given key. If no item is found, nil is returned
// and the second return value is false.
func (pm *ParMap) Get(key string) (fmt.Stringer, bool) {
	shard := pm.getShard(key)
	shard.mutex.RLock()
	item, ok := shard.index[key]
	shard.mutex.RUnlock()

	return item, ok
}

// Count returns the total number of items in the map.
func (pm *ParMap) Count() int {
	count := 0
	for _, shard := range pm.shards {
		shard.mutex.RLock()
		count += len(shard.index)
		shard.mutex.RUnlock()
	}

	return count
}

// Has checks whether a certain item is present in the map.
func (pm *ParMap) Has(item fmt.Stringer) bool {
	key := item.String()
	return pm.HasKey(key)
}

// HasKey checks whether a certain key is present in the map.
func (pm *ParMap) HasKey(key string) bool {
	shard := pm.getShard(key)
	shard.mutex.RLock()
	_, ok := shard.index[key]
	shard.mutex.RUnlock()

	return ok
}

// Remove removes an item from the map, if present.
func (pm *ParMap) Remove(item fmt.Stringer) {
	key := item.String()
	pm.RemoveKey(key)
}

// RemoveKey removes the item for a key from the map, if present.
func (pm *ParMap) RemoveKey(key string) {
	shard := pm.getShard(key)
	shard.mutex.Lock()
	delete(shard.index, key)
	shard.mutex.Unlock()
}

// Iter returns a channel that allows us to range over the map similarly to
// how we range over normal hash maps. In order to do so, we need to create
// a sub-routine, though.
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

type shard struct {
	index map[string]fmt.Stringer
	mutex *sync.RWMutex
}

func newShard() *shard {
	shard := &shard{
		index: make(map[string]fmt.Stringer),
		mutex: &sync.RWMutex{},
	}

	return shard
}

func (pm *ParMap) getShard(key string) *shard {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	shard := pm.shards[hasher.Sum32()%pm.count]

	return shard
}
