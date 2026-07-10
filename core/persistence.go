package core

import (
	"encoding/gob"
	"os"
	"time"
)

// dumpFile is the on-disk snapshot path. It's a var (not a const) so
// tests can point it at a temp file instead of the real working directory.
var dumpFile = "dump.cachedb"

// snapshotEntry mirrors entry with exported fields so gob can serialize it.
type snapshotEntry struct {
	Value     string
	ExpiresAt time.Time
}

// Save writes a snapshot of all non-expired keys to disk.
func Save() error {
	store.Lock()
	snapshot := make(map[string]snapshotEntry, len(store.data))
	for k, e := range store.data {
		if e.expired() {
			continue
		}
		snapshot[k] = snapshotEntry{Value: e.value, ExpiresAt: e.expiresAt}
	}
	store.Unlock()

	f, err := os.Create(dumpFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(snapshot)
}

// Load restores the keyspace from the snapshot file, if one exists. A
// missing file is not an error - it just means there's nothing to load yet.
func Load() error {
	f, err := os.Open(dumpFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	var snapshot map[string]snapshotEntry
	if err := gob.NewDecoder(f).Decode(&snapshot); err != nil {
		return err
	}

	store.Lock()
	defer store.Unlock()
	for k, se := range snapshot {
		e := entry{value: se.Value, expiresAt: se.ExpiresAt}
		if e.expired() {
			continue
		}
		store.data[k] = e
	}
	return nil
}
