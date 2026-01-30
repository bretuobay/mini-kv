package minikv

import (
	"time"

	"github.com/bretuobay/mini-kv/internal/wal"
)

// Delete removes a key if it exists.
func (db *DB) Delete(key []byte) error {
	stats := db.statsOrInit()
	start := time.Now()
	if len(key) > db.opts.MaxKeySize {
		stats.deletes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrKeyTooLarge
	}
	if len(key) == 0 {
		stats.deletes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		stats.deletes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrClosed
	}
	if db.opts.ReadOnly {
		stats.deletes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return ErrReadOnly
	}

	record := wal.WALRecord{
		Type:      wal.RecordDelete,
		Timestamp: time.Now().UnixNano(),
		ExpiresAt: -1,
		Key:       append([]byte(nil), key...),
	}
	if err := db.wal.AppendRecord(record); err != nil {
		stats.deletes.Add(1)
		stats.writeLatency.add(time.Since(start))
		return err
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			stats.deletes.Add(1)
			stats.writeLatency.add(time.Since(start))
			return err
		}
	}

	db.index.Delete(string(key))
	stats.deletes.Add(1)
	stats.writeLatency.add(time.Since(start))
	return nil
}
