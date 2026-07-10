package core

import (
	"testing"
	"time"

	"github.com/nahid12105080/cacheDB/config"
)

func withMaxKeys(t *testing.T, n int) {
	t.Helper()
	original := config.MaxKeys
	config.MaxKeys = n
	t.Cleanup(func() { config.MaxKeys = original })
}

func TestEvictionDisabledByDefault(t *testing.T) {
	resetStore(t)
	withMaxKeys(t, 0)

	for i := 0; i < 10; i++ {
		setKey(string(rune('a'+i)), "value")
	}

	if got := len(store.data); got != 10 {
		t.Fatalf("expected all 10 keys to be retained when MaxKeys is 0, got %d", got)
	}
}

func TestEvictionCapsKeyCount(t *testing.T) {
	resetStore(t)
	withMaxKeys(t, 3)

	setKey("a", "1")
	setKey("b", "2")
	setKey("c", "3")
	setKey("d", "4")

	if got := len(store.data); got != 3 {
		t.Fatalf("expected store to be capped at 3 keys, got %d", got)
	}
}

func TestEvictionPrefersSoonestExpiry(t *testing.T) {
	resetStore(t)
	withMaxKeys(t, 2)

	setKeyWithTTL("soon", "value", 10*time.Second)
	setKey("persistent", "value")
	// Inserting a third key should evict "soon", since it's the only key
	// with a TTL and eviction prefers soonest-to-expire.
	setKey("newcomer", "value")

	if _, ok := getKey("soon"); ok {
		t.Fatalf("expected the key with a TTL to be evicted first")
	}
	if _, ok := getKey("persistent"); !ok {
		t.Fatalf("expected 'persistent' key to survive eviction")
	}
	if _, ok := getKey("newcomer"); !ok {
		t.Fatalf("expected 'newcomer' key to survive eviction")
	}
}

func TestEvictionUpdatingExistingKeyDoesNotEvict(t *testing.T) {
	resetStore(t)
	withMaxKeys(t, 2)

	setKey("a", "1")
	setKey("b", "2")
	setKey("a", "updated")

	if got := len(store.data); got != 2 {
		t.Fatalf("expected key count to stay at 2 when updating an existing key, got %d", got)
	}
	val, ok := getKey("a")
	if !ok || val != "updated" {
		t.Fatalf("expected 'a' to be updated in place, got %q ok=%v", val, ok)
	}
}
