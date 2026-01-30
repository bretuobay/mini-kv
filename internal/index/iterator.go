package index

import (
	"sort"
	"strings"
	"time"
)

// KeyEntry bundles a key with its entry data.
type KeyEntry struct {
	Key   []byte
	Entry Entry
}

// Scan returns up to limit entries whose keys have the given prefix.
func (m *MemIndex) Scan(prefix string, limit int) []KeyEntry {
	return m.scan(func(key string) bool { return strings.HasPrefix(key, prefix) }, limit)
}

// ScanRange returns up to limit entries whose keys are in [start, end].
func (m *MemIndex) ScanRange(start, end string, limit int) []KeyEntry {
	return m.scan(func(key string) bool { return key >= start && key <= end }, limit)
}

// Keys returns all keys matching the glob pattern (supports '*' and '?').
func (m *MemIndex) Keys(pattern string) []string {
	matches := m.scanKeys(func(key string) bool {
		ok, _ := pathMatch(pattern, key)
		return ok
	})
	return matches
}

func (m *MemIndex) scan(match func(string) bool, limit int) []KeyEntry {
	now := time.Now().UnixNano()

	m.mu.RLock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		if match(k) {
			keys = append(keys, k)
		}
	}
	m.mu.RUnlock()

	sort.Strings(keys)
	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}

	results := make([]KeyEntry, 0, len(keys))
	for _, key := range keys {
		m.mu.RLock()
		entry, ok := m.data[key]
		m.mu.RUnlock()
		if !ok {
			continue
		}
		if isExpired(entry.ExpiresAt, now) {
			m.mu.Lock()
			entry, ok = m.data[key]
			if ok && isExpired(entry.ExpiresAt, now) {
				delete(m.data, key)
				m.size -= entrySize(key, entry)
			}
			m.mu.Unlock()
			continue
		}
		results = append(results, KeyEntry{
			Key: cloneBytes([]byte(key)),
			Entry: Entry{
				Value:     cloneBytes(entry.Value),
				ExpiresAt: entry.ExpiresAt,
				CreatedAt: entry.CreatedAt,
			},
		})
	}
	return results
}

func (m *MemIndex) scanKeys(match func(string) bool) []string {
	m.mu.RLock()
	keys := make([]string, 0, len(m.data))
	for k, entry := range m.data {
		if isExpired(entry.ExpiresAt, time.Now().UnixNano()) {
			continue
		}
		if match(k) {
			keys = append(keys, k)
		}
	}
	m.mu.RUnlock()

	sort.Strings(keys)
	return keys
}
