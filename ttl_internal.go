package minikv

import (
	"time"

	"mini-kv/internal/wal"
)

func (db *DB) setWithExpiresAt(key []byte, value []byte, expiresAt int64) error {
	if len(key) > db.opts.MaxKeySize {
		return ErrKeyTooLarge
	}
	if len(value) > db.opts.MaxValueSize {
		return ErrValueTooLarge
	}
	if len(key) == 0 {
		return ErrNotFound
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}
	if db.opts.ReadOnly {
		return ErrReadOnly
	}

	now := time.Now().UnixNano()
	record := wal.WALRecord{
		Type:      wal.RecordSet,
		Timestamp: now,
		ExpiresAt: expiresAt,
		Key:       append([]byte(nil), key...),
		Value:     append([]byte(nil), value...),
	}
	if err := db.wal.AppendRecord(record); err != nil {
		return err
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			return err
		}
	}

	db.index.SetEntry(string(key), value, expiresAt, now)
	return nil
}

func (db *DB) updateExpiresAtLocked(key []byte, value []byte, expiresAt int64, createdAt int64) (bool, error) {
	now := time.Now().UnixNano()
	record := wal.WALRecord{
		Type:      wal.RecordSet,
		Timestamp: now,
		ExpiresAt: expiresAt,
		Key:       append([]byte(nil), key...),
		Value:     append([]byte(nil), value...),
	}
	if err := db.wal.AppendRecord(record); err != nil {
		return false, err
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			return false, err
		}
	}

	db.index.SetEntry(string(key), value, expiresAt, createdAt)
	return true, nil
}
