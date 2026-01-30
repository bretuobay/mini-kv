package minikv

// Exists reports whether a key exists and is not expired.
func (db *DB) Exists(key []byte) (bool, error) {
	if len(key) > db.opts.MaxKeySize {
		return false, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return false, nil
	}

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return false, ErrClosed
	}
	_, ok := db.index.Get(string(key))
	db.mu.RUnlock()
	return ok, nil
}
