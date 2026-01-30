package minikv

import "time"

// Get returns the value for a key or ErrNotFound.
func (db *DB) Get(key []byte) ([]byte, error) {
	if len(key) > db.opts.MaxKeySize {
		return nil, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return nil, ErrNotFound
	}

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return nil, ErrClosed
	}
	entry, ok := db.index.Get(string(key))
	db.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if entry.ExpiresAt >= 0 && entry.ExpiresAt <= time.Now().UnixNano() {
		return nil, ErrNotFound
	}
	value := make([]byte, len(entry.Value))
	copy(value, entry.Value)
	return value, nil
}
