package minikv

import "time"

// SetWithTTL stores a key-value pair with a TTL duration.
func (db *DB) SetWithTTL(key []byte, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return db.Set(key, value)
	}
	expiresAt := time.Now().Add(ttl).UnixNano()
	return db.setWithExpiresAt(key, value, expiresAt)
}

// TTL returns the remaining TTL for a key, or ErrNotFound.
// Returns -1 for keys without expiration.
func (db *DB) TTL(key []byte) (time.Duration, error) {
	if len(key) > db.opts.MaxKeySize {
		return 0, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return 0, ErrNotFound
	}

	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return 0, ErrClosed
	}
	entry, ok := db.index.Get(string(key))
	db.mu.RUnlock()
	if !ok {
		return 0, ErrNotFound
	}
	if entry.ExpiresAt < 0 {
		return -1, nil
	}

	now := time.Now().UnixNano()
	if entry.ExpiresAt <= now {
		return 0, ErrNotFound
	}
	return time.Duration(entry.ExpiresAt-now) * time.Nanosecond, nil
}

// Expire sets a TTL on an existing key.
func (db *DB) Expire(key []byte, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, nil
	}
	if len(key) > db.opts.MaxKeySize {
		return false, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return false, nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return false, ErrClosed
	}
	if db.opts.ReadOnly {
		return false, ErrReadOnly
	}

	entry, ok := db.index.Get(string(key))
	if !ok {
		return false, nil
	}

	expiresAt := time.Now().Add(ttl).UnixNano()
	return db.updateExpiresAtLocked(key, entry.Value, expiresAt, entry.CreatedAt)
}

// Persist removes expiration from an existing key.
func (db *DB) Persist(key []byte) (bool, error) {
	if len(key) > db.opts.MaxKeySize {
		return false, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return false, nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return false, ErrClosed
	}
	if db.opts.ReadOnly {
		return false, ErrReadOnly
	}

	entry, ok := db.index.Get(string(key))
	if !ok {
		return false, nil
	}

	return db.updateExpiresAtLocked(key, entry.Value, -1, entry.CreatedAt)
}
