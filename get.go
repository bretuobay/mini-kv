package minikv

import "time"

// Get returns the value for a key or ErrNotFound.
func (db *DB) Get(key []byte) ([]byte, error) {
	stats := db.statsOrInit()
	start := time.Now()
	if len(key) > db.opts.MaxKeySize {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrKeyTooLarge
	}
	if len(key) == 0 {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrNotFound
	}

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrClosed
	}
	entry, ok := db.index.Get(string(key))
	db.mu.RUnlock()
	if !ok {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrNotFound
	}
	if entry.ExpiresAt >= 0 && entry.ExpiresAt <= time.Now().UnixNano() {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrNotFound
	}
	value := make([]byte, len(entry.Value))
	copy(value, entry.Value)
	stats.reads.Add(1)
	stats.readLatency.add(time.Since(start))
	return value, nil
}
