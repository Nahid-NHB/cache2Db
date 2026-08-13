package core

import (
	"math"
	"sync"
	"time"

	"github.com/nahid12105080/cacheDB/config"
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
	evictIfNeeded()
}

func setKeyWithTTL(k, v string, ttl time.Duration) {
	store.Lock()
	defer store.Unlock()
	store.data[k] = entry{value: v, expiresAt: time.Now().Add(ttl)}
	evictIfNeeded()
}

// evictIfNeeded removes one key if the store is over config.MaxKeys. It
// must be called with the store lock already held. Preference is given to
// evicting the key with the soonest expiry; if no key has a TTL, an
// arbitrary key is evicted instead (Go map iteration order is randomized).
func evictIfNeeded() {
	if config.MaxKeys <= 0 || len(store.data) <= config.MaxKeys {
		return
	}

	var victim string
	var soonest time.Time
	found := false

	for k, e := range store.data {
		if e.expiresAt.IsZero() {
			continue
		}
		if !found || e.expiresAt.Before(soonest) {
			victim = k
			soonest = e.expiresAt
			found = true
		}
	}

	if !found {
		for k := range store.data {
			victim = k
			break
		}
	}

	delete(store.data, victim)
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

// keysMatching returns all non-expired keys matching a glob pattern.
// Pattern supports * (any chars), ? (single char), [abc] (one of).
func keysMatching(pattern string) []string {
	store.RLock()
	defer store.RUnlock()

	var result []string
	for k, e := range store.data {
		if e.expired() {
			continue
		}
		if matchGlob(k, pattern) {
			result = append(result, k)
		}
	}
	return result
}

// dbsize returns the count of non-expired keys in the store.
func dbsize() int {
	store.RLock()
	defer store.RUnlock()

	count := 0
	for _, e := range store.data {
		if !e.expired() {
			count++
		}
	}
	return count
}

// matchGlob matches a string against a glob pattern.
// Supports: * (any sequence), ? (single char), [abc] (char class)
func matchGlob(s, pattern string) bool {
	return globMatch(s, pattern, 0, 0)
}

func globMatch(s, pattern string, si, pi int) bool {
	if pi == len(pattern) {
		return si == len(s)
	}

	switch pattern[pi] {
	case '*':
		if pi+1 == len(pattern) {
			return true
		}
		for i := si; i <= len(s); i++ {
			if globMatch(s, pattern, i, pi+1) {
				return true
			}
		}
		return false

	case '?':
		if si >= len(s) {
			return false
		}
		return globMatch(s, pattern, si+1, pi+1)

	case '[':
		if si >= len(s) {
			return false
		}
		pj := pi + 1
		negate := false
		if pj < len(pattern) && pattern[pj] == '^' {
			negate = true
			pj++
		}
		found := false
		for pj < len(pattern) && pattern[pj] != ']' {
			if pattern[pj] == s[si] {
				found = true
			}
			pj++
		}
		if negate {
			found = !found
		}
		if !found {
			return false
		}
		return globMatch(s, pattern, si+1, pj+1)

	default:
		if si >= len(s) || s[si] != pattern[pi] {
			return false
		}
		return globMatch(s, pattern, si+1, pi+1)
	}
}
