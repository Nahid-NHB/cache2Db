package core

import (
	"testing"
	"time"
)

func TestSetGetKey(t *testing.T) {
	setKey("foo", "bar")

	val, ok := getKey("foo")
	if !ok {
		t.Fatalf("expected key 'foo' to exist")
	}
	if val != "bar" {
		t.Fatalf("expected 'bar', got %q", val)
	}
}

func TestGetKeyMissing(t *testing.T) {
	_, ok := getKey("does-not-exist")
	if ok {
		t.Fatalf("expected key to not exist")
	}
}

func TestDeleteKey(t *testing.T) {
	setKey("todelete", "value")

	if !deleteKey("todelete") {
		t.Fatalf("expected deleteKey to return true for existing key")
	}
	if deleteKey("todelete") {
		t.Fatalf("expected deleteKey to return false for already-deleted key")
	}

	_, ok := getKey("todelete")
	if ok {
		t.Fatalf("expected key to be gone after delete")
	}
}

func TestTTLSecondsNoExpiry(t *testing.T) {
	setKey("noexpiry", "value")

	if ttl := ttlSeconds("noexpiry"); ttl != -1 {
		t.Fatalf("expected -1 for key with no expiry, got %d", ttl)
	}
}

func TestTTLSecondsMissingKey(t *testing.T) {
	if ttl := ttlSeconds("does-not-exist"); ttl != -2 {
		t.Fatalf("expected -2 for missing key, got %d", ttl)
	}
}

func TestSetKeyWithTTLAndPassiveExpiration(t *testing.T) {
	setKeyWithTTL("expiring", "value", 20*time.Millisecond)

	val, ok := getKey("expiring")
	if !ok || val != "value" {
		t.Fatalf("expected key to be readable before expiry")
	}

	if ttl := ttlSeconds("expiring"); ttl <= 0 {
		t.Fatalf("expected positive ttl before expiry, got %d", ttl)
	}

	time.Sleep(30 * time.Millisecond)

	if _, ok := getKey("expiring"); ok {
		t.Fatalf("expected key to be gone after expiry")
	}
}

func TestExpireKey(t *testing.T) {
	setKey("tobeexpired", "value")

	if !expireKey("tobeexpired", 20*time.Millisecond) {
		t.Fatalf("expected expireKey to return true for existing key")
	}

	if expireKey("missing", 20*time.Millisecond) {
		t.Fatalf("expected expireKey to return false for missing key")
	}

	time.Sleep(30 * time.Millisecond)

	if _, ok := getKey("tobeexpired"); ok {
		t.Fatalf("expected key to be gone after expiry")
	}
}
