package core

import (
	"path/filepath"
	"testing"
	"time"
)

func withTempDumpFile(t *testing.T) {
	t.Helper()
	original := dumpFile
	dumpFile = filepath.Join(t.TempDir(), "dump.cachedb")
	t.Cleanup(func() { dumpFile = original })
}

func resetStore(t *testing.T) {
	t.Helper()
	store.Lock()
	store.data = make(map[string]entry)
	store.Unlock()
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	withTempDumpFile(t)
	resetStore(t)

	setKey("persisted", "value")
	setKeyWithTTL("withttl", "ttlvalue", time.Hour)

	if err := Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	resetStore(t)

	if err := Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	val, ok := getKey("persisted")
	if !ok || val != "value" {
		t.Fatalf("expected 'persisted' key to survive round trip, got %q ok=%v", val, ok)
	}

	val, ok = getKey("withttl")
	if !ok || val != "ttlvalue" {
		t.Fatalf("expected 'withttl' key to survive round trip, got %q ok=%v", val, ok)
	}

	if ttl := ttlSeconds("withttl"); ttl <= 0 {
		t.Fatalf("expected TTL to survive round trip, got %d", ttl)
	}
}

func TestSaveExcludesExpiredKeys(t *testing.T) {
	withTempDumpFile(t)
	resetStore(t)

	setKeyWithTTL("expiring", "value", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if err := Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	resetStore(t)

	if err := Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if _, ok := getKey("expiring"); ok {
		t.Fatalf("expected expired key to be excluded from snapshot")
	}
}

func TestLoadMissingFileIsNoOp(t *testing.T) {
	withTempDumpFile(t)
	resetStore(t)

	if err := Load(); err != nil {
		t.Fatalf("expected Load to be a no-op for a missing file, got: %v", err)
	}
}
