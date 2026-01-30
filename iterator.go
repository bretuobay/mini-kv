package minikv

import "time"

// Scan returns up to limit key/value pairs matching prefix in lexicographic order.
func (db *DB) Scan(prefix []byte, limit int) ([][]byte, [][]byte, error) {
	stats := db.statsOrInit()
	start := time.Now()
	if len(prefix) > db.opts.MaxKeySize {
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, nil, ErrKeyTooLarge
	}
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, nil, ErrClosed
	}
	entries := db.index.Scan(string(prefix), limit)
	db.mu.RUnlock()

	keys := make([][]byte, 0, len(entries))
	values := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
		values = append(values, entry.Entry.Value)
	}
	stats.scans.Add(1)
	stats.readLatency.add(time.Since(start))
	return keys, values, nil
}

// ScanRange returns up to limit key/value pairs whose keys are within [start, end].
func (db *DB) ScanRange(start, end []byte, limit int) ([][]byte, [][]byte, error) {
	stats := db.statsOrInit()
	startTime := time.Now()
	if len(start) > db.opts.MaxKeySize || len(end) > db.opts.MaxKeySize {
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(startTime))
		return nil, nil, ErrKeyTooLarge
	}
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(startTime))
		return nil, nil, ErrClosed
	}
	entries := db.index.ScanRange(string(start), string(end), limit)
	db.mu.RUnlock()

	keys := make([][]byte, 0, len(entries))
	values := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
		values = append(values, entry.Entry.Value)
	}
	stats.scans.Add(1)
	stats.readLatency.add(time.Since(startTime))
	return keys, values, nil
}

// Keys returns keys matching a glob pattern.
func (db *DB) Keys(pattern string) ([]string, error) {
	stats := db.statsOrInit()
	start := time.Now()
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(start))
		return nil, ErrClosed
	}
	keys := db.index.Keys(pattern)
	db.mu.RUnlock()
	stats.scans.Add(1)
	stats.readLatency.add(time.Since(start))
	return keys, nil
}

// Count returns the total number of non-expired keys.
func (db *DB) Count() (int, error) {
	stats := db.statsOrInit()
	start := time.Now()
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		stats.scans.Add(1)
		stats.readLatency.add(time.Since(start))
		return 0, ErrClosed
	}
	count := db.index.Count()
	db.mu.RUnlock()
	stats.scans.Add(1)
	stats.readLatency.add(time.Since(start))
	return count, nil
}
