package core

import (
	"math"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time // zero value means no expiry
}

func (e entry) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

var store = struct {
	sync.RWMutex
	data map[string]entry
}{data: make(map[string]entry)}

func setKey(k, v string) {
	store.Lock()
	defer store.Unlock()
	store.data[k] = entry{value: v}
}

func setKeyWithTTL(k, v string, ttl time.Duration) {
	store.Lock()
	defer store.Unlock()
	store.data[k] = entry{value: v, expiresAt: time.Now().Add(ttl)}
}

func getKey(k string) (string, bool) {
	store.Lock()
	defer store.Unlock()

	e, ok := store.data[k]
	if !ok {
		return "", false
	}
	if e.expired() {
		delete(store.data, k)
		return "", false
	}
	return e.value, true
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

// expireKey sets a new TTL on an existing, non-expired key. It reports
// whether the key existed.
func expireKey(k string, ttl time.Duration) bool {
	store.Lock()
	defer store.Unlock()

	e, ok := store.data[k]
	if !ok || e.expired() {
		delete(store.data, k)
		return false
	}

	e.expiresAt = time.Now().Add(ttl)
	store.data[k] = e
	return true
}

// ttlSeconds reports the remaining TTL for a key in seconds: -2 if the key
// doesn't exist (or has expired), -1 if it exists but has no expiry, else
// the number of seconds remaining, rounded up.
func ttlSeconds(k string) int64 {
	store.Lock()
	defer store.Unlock()

	e, ok := store.data[k]
	if !ok {
		return -2
	}
	if e.expired() {
		delete(store.data, k)
		return -2
	}
	if e.expiresAt.IsZero() {
		return -1
	}

	return int64(math.Ceil(time.Until(e.expiresAt).Seconds()))
}
