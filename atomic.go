package minikv

import (
	"bytes"
	"strconv"
	"time"
)

// SetNX sets the value only if the key does not exist.
func (db *DB) SetNX(key []byte, value []byte) (bool, error) {
	if len(key) > db.opts.MaxKeySize {
		return false, ErrKeyTooLarge
	}
	if len(value) > db.opts.MaxValueSize {
		return false, ErrValueTooLarge
	}
	if len(key) == 0 {
		return false, ErrNotFound
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return false, ErrClosed
	}
	if db.opts.ReadOnly {
		return false, ErrReadOnly
	}
	if _, ok := db.index.Get(string(key)); ok {
		return false, nil
	}
	if err := db.setWithExpiresAtLocked(key, value, -1, 0, false); err != nil {
		return false, err
	}
	return true, nil
}

// Incr increments the integer value by 1.
func (db *DB) Incr(key []byte) (int64, error) {
	return db.IncrBy(key, 1)
}

// Decr decrements the integer value by 1.
func (db *DB) Decr(key []byte) (int64, error) {
	return db.IncrBy(key, -1)
}

// IncrBy increments the integer value by delta.
func (db *DB) IncrBy(key []byte, delta int64) (int64, error) {
	if len(key) > db.opts.MaxKeySize {
		return 0, ErrKeyTooLarge
	}
	if len(key) == 0 {
		return 0, ErrNotFound
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return 0, ErrClosed
	}
	if db.opts.ReadOnly {
		return 0, ErrReadOnly
	}

	entry, ok := db.index.Get(string(key))
	var current int64
	var createdAt int64
	if !ok {
		current = 0
		createdAt = time.Now().UnixNano()
	} else {
		parsed, err := strconv.ParseInt(string(entry.Value), 10, 64)
		if err != nil {
			return 0, ErrInvalidValue
		}
		current = parsed
		createdAt = entry.CreatedAt
	}
	newVal := current + delta
	valueBytes := []byte(strconv.FormatInt(newVal, 10))

	if err := db.setWithExpiresAtLocked(key, valueBytes, -1, createdAt, true); err != nil {
		return 0, err
	}
	return newVal, nil
}

// CompareAndSwap sets new value if current value matches old.
func (db *DB) CompareAndSwap(key []byte, oldVal []byte, newVal []byte) (bool, error) {
	if len(key) > db.opts.MaxKeySize {
		return false, ErrKeyTooLarge
	}
	if len(newVal) > db.opts.MaxValueSize {
		return false, ErrValueTooLarge
	}
	if len(key) == 0 {
		return false, ErrNotFound
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
	if !bytes.Equal(entry.Value, oldVal) {
		return false, nil
	}
	if err := db.setWithExpiresAtLocked(key, newVal, entry.ExpiresAt, entry.CreatedAt, true); err != nil {
		return false, err
	}
	return true, nil
}

// GetAndSet atomically sets new value and returns the old value.
func (db *DB) GetAndSet(key []byte, value []byte) ([]byte, error) {
	if len(key) > db.opts.MaxKeySize {
		return nil, ErrKeyTooLarge
	}
	if len(value) > db.opts.MaxValueSize {
		return nil, ErrValueTooLarge
	}
	if len(key) == 0 {
		return nil, ErrNotFound
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return nil, ErrClosed
	}
	if db.opts.ReadOnly {
		return nil, ErrReadOnly
	}

	entry, ok := db.index.Get(string(key))
	var old []byte
	if ok {
		old = append([]byte(nil), entry.Value...)
	}
	if err := db.setWithExpiresAtLocked(key, value, -1, 0, false); err != nil {
		return nil, err
	}
	return old, nil
}
