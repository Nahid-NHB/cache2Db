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

func TestMatchGlobAsterisk(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"foo", "f*", true},
		{"foobar", "f*", true},
		{"foobar", "foo*", true},
		{"foobar", "*bar", true},
		{"foobar", "*o*", true},
		{"bar", "f*", false},
		{"", "*", true},
		{"x", "*", true},
	}

	for _, tt := range tests {
		if got := matchGlob(tt.s, tt.pattern); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchGlobQuestion(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"a", "?", true},
		{"ab", "a?", true},
		{"abc", "a?c", true},
		{"ab", "a?c", false},
		{"", "?", false},
	}

	for _, tt := range tests {
		if got := matchGlob(tt.s, tt.pattern); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchGlobCharClass(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"a", "[abc]", true},
		{"b", "[abc]", true},
		{"d", "[abc]", false},
		{"a1", "[abc]1", true},
		{"d1", "[abc]1", false},
	}

	for _, tt := range tests {
		if got := matchGlob(tt.s, tt.pattern); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
		}
	}
}

func TestKeysMatching(t *testing.T) {
	resetStore(t)
	setKey("foo", "1")
	setKey("foobar", "2")
	setKey("bar", "3")
	setKeyWithTTL("expiring", "4", 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	keys := keysMatching("*")
	if len(keys) != 3 {
		t.Fatalf("expected 3 non-expired keys, got %d: %v", len(keys), keys)
	}

	keys = keysMatching("foo*")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys matching 'foo*', got %d: %v", len(keys), keys)
	}
}

func TestDBSize(t *testing.T) {
	resetStore(t)
	if dbsize() != 0 {
		t.Fatalf("expected empty store to have size 0")
	}

	setKey("a", "1")
	setKey("b", "2")
	if dbsize() != 2 {
		t.Fatalf("expected size 2, got %d", dbsize())
	}

	setKeyWithTTL("c", "3", 10*time.Millisecond)
	if dbsize() != 3 {
		t.Fatalf("expected size 3 before expiry, got %d", dbsize())
	}

	time.Sleep(20 * time.Millisecond)
	if dbsize() != 2 {
		t.Fatalf("expected size 2 after expiry, got %d", dbsize())
	}
}
