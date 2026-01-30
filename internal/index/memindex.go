package index

import (
	"sync"
	"time"
)

// Entry represents a key-value pair with metadata.
type Entry struct {
	Value     []byte
	ExpiresAt int64
	CreatedAt int64
}

// MemIndex is the in-memory key-value index.
type MemIndex struct {
	mu   sync.RWMutex
	data map[string]*Entry
	size int64
}

// NewMemIndex creates an empty in-memory index.
func NewMemIndex() *MemIndex {
	return &MemIndex{data: make(map[string]*Entry)}
}

// Set stores a key with value and expiration timestamp (Unix nanoseconds).
// Use expiresAt = -1 to indicate no expiration.
func (m *MemIndex) Set(key string, value []byte, expiresAt int64) {
	m.SetEntry(key, value, expiresAt, time.Now().UnixNano())
}

// SetEntry stores a key with explicit creation timestamp.
func (m *MemIndex) SetEntry(key string, value []byte, expiresAt int64, createdAt int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.data[key]; ok {
		m.size -= entrySize(key, existing)
	}

	entry := &Entry{
		Value:     cloneBytes(value),
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}
	m.data[key] = entry
	m.size += entrySize(key, entry)
}

// Get returns the entry for key if it exists and is not expired.
func (m *MemIndex) Get(key string) (*Entry, bool) {
	m.mu.RLock()
	entry, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if isExpired(entry.ExpiresAt, time.Now().UnixNano()) {
		m.mu.Lock()
		// Recheck under write lock before delete.
		entry, ok = m.data[key]
		if ok && isExpired(entry.ExpiresAt, time.Now().UnixNano()) {
			delete(m.data, key)
			m.size -= entrySize(key, entry)
		}
		m.mu.Unlock()
		return nil, false
	}

	return entry, true
}

// Delete removes a key if present.
func (m *MemIndex) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, ok := m.data[key]; ok {
		delete(m.data, key)
		m.size -= entrySize(key, entry)
	}
}

// Exists reports whether key exists and is not expired.
func (m *MemIndex) Exists(key string) bool {
	_, ok := m.Get(key)
	return ok
}

// Count returns the number of non-expired keys.
func (m *MemIndex) Count() int {
	now := time.Now().UnixNano()

	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for k, entry := range m.data {
		if isExpired(entry.ExpiresAt, now) {
			delete(m.data, k)
			m.size -= entrySize(k, entry)
			continue
		}
		count++
	}

	return count
}

// Size returns the estimated memory size in bytes.
func (m *MemIndex) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.size
}

func isExpired(expiresAt int64, now int64) bool {
	if expiresAt < 0 {
		return false
	}
	if expiresAt == 0 {
		return false
	}
	return now >= expiresAt
}

func entrySize(key string, entry *Entry) int64 {
	if entry == nil {
		return 0
	}
	return int64(len(key) + len(entry.Value))
}

func cloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
