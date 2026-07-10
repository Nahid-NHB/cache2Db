package core

import "testing"

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
