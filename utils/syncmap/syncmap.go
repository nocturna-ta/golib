package syncmap

import (
	"github.com/nocturna-ta/golib/utils/maps"
	"sync"
)

type SyncMap[T any] struct {
	sync.RWMutex
	internal map[string]T
}

func NewSyncMap[T any]() *SyncMap[T] {
	return &SyncMap[T]{
		internal: make(map[string]T),
	}
}

func (rm *SyncMap[T]) Get(key string) T {
	rm.RLock()
	result := rm.internal[key]
	rm.RUnlock()
	return result
}

func (rm *SyncMap[T]) Delete(key string) {
	rm.Lock()
	delete(rm.internal, key)
	rm.Unlock()
}

func (rm *SyncMap[T]) Store(key string, value T) {
	rm.Lock()
	rm.internal[key] = value
	rm.Unlock()
}

func (rm *SyncMap[T]) GetAllKey() []string {
	rm.Lock()
	keys := make([]string, len(rm.internal))
	i := 0
	for k := range rm.internal {
		keys[i] = k
		i++
	}
	rm.Unlock()

	return keys
}

func (rm *SyncMap[T]) GetAllValues() []T {
	return maps.Values(rm.internal)
}
