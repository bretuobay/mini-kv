package minikv

import (
	"time"

	"mini-kv/internal/wal"
)

func (db *DB) setWithExpiresAt(key []byte, value []byte, expiresAt int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.setWithExpiresAtLocked(key, value, expiresAt, 0, false)
}

func (db *DB) setWithExpiresAtLocked(key []byte, value []byte, expiresAt int64, createdAt int64, preserveCreated bool) error {
	stats := db.statsOrInit()
	start := time.Now()
	if len(key) > db.opts.MaxKeySize {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrKeyTooLarge
	}
	if len(value) > db.opts.MaxValueSize {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrValueTooLarge
	}
	if len(key) == 0 {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrNotFound
	}

	if db.closed {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrClosed
	}
	if db.opts.ReadOnly {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrReadOnly
	}

	now := time.Now().UnixNano()
	if !preserveCreated || createdAt == 0 {
		createdAt = now
	}

	record := wal.WALRecord{
		Type:      wal.RecordSet,
		Timestamp: now,
		ExpiresAt: expiresAt,
		Key:       append([]byte(nil), key...),
		Value:     append([]byte(nil), value...),
	}
	if err := db.wal.AppendRecord(record); err != nil {
		stats.writes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return err
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			stats.writes.Add(1)
			stats.writeLatency.add(time.Since(start))
			return err
		}
	}

	db.index.SetEntry(string(key), value, expiresAt, createdAt)
	stats.writes.Add(1)
	stats.writeLatency.add(time.Since(start))
	return nil
}

func (db *DB) updateExpiresAtLocked(key []byte, value []byte, expiresAt int64, createdAt int64) (bool, error) {
	if err := db.setWithExpiresAtLocked(key, value, expiresAt, createdAt, true); err != nil {
		return false, err
	}
	return true, nil
}
