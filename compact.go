package minikv

import (
	"os"
	"path/filepath"
	"time"

	"github.com/bretuobay/mini-kv/internal/snapshot"
	"github.com/bretuobay/mini-kv/internal/wal"
)

// Compact creates a snapshot and removes old WAL segments.
func (db *DB) Compact() error {
	if !db.beginCompaction() {
		return nil
	}
	defer db.endCompaction()

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return ErrClosed
	}
	entries := db.index.Scan("", 0)
	seq := db.wal.CurrentSeq()
	snapMgr := db.snap
	db.mu.RUnlock()

	now := time.Now().UnixNano()
	snapEntries := make([]snapshot.Entry, 0, len(entries))
	for _, entry := range entries {
		snapEntries = append(snapEntries, snapshot.Entry{
			Key:       entry.Key,
			Value:     entry.Entry.Value,
			ExpiresAt: entry.Entry.ExpiresAt,
			CreatedAt: entry.Entry.CreatedAt,
		})
	}

	if seq == 0 {
		seq = 1
	}
	_, err := snapMgr.CreateSnapshot(snapEntries, 1, now, seq)
	if err != nil {
		return err
	}

	if err := deleteOldWALSegments(filepath.Join(db.path, "wal"), seq); err != nil {
		return err
	}

	return refreshManifest(db.path)
}

func (db *DB) beginCompaction() bool {
	db.compactMu.Lock()
	defer db.compactMu.Unlock()
	if db.compacting {
		return false
	}
	db.compacting = true
	return true
}

func (db *DB) endCompaction() {
	db.compactMu.Lock()
	db.compacting = false
	db.compactMu.Unlock()
}

func (db *DB) compactAsync() {
	go func() {
		_ = db.Compact()
	}()
}

func deleteOldWALSegments(walDir string, keepSeq uint64) error {
	segments, err := wal.ListSegments(walDir)
	if err != nil {
		return err
	}
	for _, path := range segments {
		seq, ok := parseSegmentSeq(path)
		if !ok {
			continue
		}
		if seq < keepSeq {
			_ = os.Remove(path)
		}
	}
	return nil
}
