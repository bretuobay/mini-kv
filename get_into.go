package minikv

import "time"

// GetInto copies the value into dst and returns the resulting slice.
// If dst has sufficient capacity, it is reused to reduce allocations.
func (db *DB) GetInto(dst []byte, key []byte) ([]byte, error) {
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

	if cap(dst) < len(entry.Value) {
		dst = make([]byte, len(entry.Value))
	} else {
		dst = dst[:len(entry.Value)]
	}
	copy(dst, entry.Value)
	stats.reads.Add(1)
	stats.readLatency.add(time.Since(start))
	return dst, nil
}
