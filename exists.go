package minikv

import "time"

// Exists reports whether a key exists and is not expired.
func (db *DB) Exists(key []byte) (bool, error) {
	stats := db.statsOrInit()
	start := time.Now()
	if len(key) > db.opts.MaxKeySize {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return false, ErrKeyTooLarge
	}
	if len(key) == 0 {
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return false, nil
	}

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.reads.Add(1)
		stats.readLatency.add(time.Since(start))
		return false, ErrClosed
	}
	_, ok := db.index.Get(string(key))
	db.mu.RUnlock()
	stats.reads.Add(1)
	stats.readLatency.add(time.Since(start))
	return ok, nil
}
