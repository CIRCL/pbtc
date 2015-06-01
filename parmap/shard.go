package parmap

import (
	"fmt"
	"sync"
)

// shard is one synchronized hashmap used for a section of the global map
type shard struct {
	index map[string]fmt.Stringer
	mutex *sync.RWMutex
}

// newShard creates a new shard with an initialized mutex for synchronization
func newShard() *shard {
	shard := &shard{
		index: make(map[string]fmt.Stringer),
		mutex: &sync.RWMutex{},
	}

	return shard
}
