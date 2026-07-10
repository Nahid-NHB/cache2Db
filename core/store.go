package core

import "sync"

var store = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

func setKey(k, v string) {
	store.Lock()
	defer store.Unlock()
	store.data[k] = v
}

func getKey(k string) (string, bool) {
	store.RLock()
	defer store.RUnlock()
	v, ok := store.data[k]
	return v, ok
}

func deleteKey(k string) bool {
	store.Lock()
	defer store.Unlock()
	_, ok := store.data[k]
	if ok {
		delete(store.data, k)
	}
	return ok
}
