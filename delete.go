package minikv

import (
	"time"

	"mini-kv/internal/wal"
)

// Delete removes a key if it exists.
func (db *DB) Delete(key []byte) error {
	if len(key) > db.opts.MaxKeySize {
		return ErrKeyTooLarge
	}
	if len(key) == 0 {
		return nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}
	if db.opts.ReadOnly {
		return ErrReadOnly
	}

	record := wal.WALRecord{
		Type:      wal.RecordDelete,
		Timestamp: time.Now().UnixNano(),
		ExpiresAt: -1,
		Key:       append([]byte(nil), key...),
	}
	if err := db.wal.AppendRecord(record); err != nil {
		return err
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			return err
		}
	}

	db.index.Delete(string(key))
	return nil
}
