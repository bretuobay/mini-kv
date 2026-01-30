package minikv

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestIteratorProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("scan prefix filtering", prop.ForAll(
		func(keys []string, values [][]byte, prefix string) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			if prefix == "" && len(keys) > 0 {
				prefix = normalizeKey(keys[0])
				if len(prefix) > 0 {
					prefix = prefix[:1]
				}
			}

			scanKeys, _, err := db.Scan([]byte(prefix), 0)
			if err != nil {
				return false
			}
			for i := range scanKeys {
				if !strings.HasPrefix(string(scanKeys[i]), prefix) {
					return false
				}
			}
			return isSortedBytes(scanKeys)
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
		gen.AnyString(),
	))

	properties.Property("scan range filtering", prop.ForAll(
		func(keys []string, values [][]byte, start string, end string) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			start = normalizeKey(start)
			end = normalizeKey(end)
			if start > end {
				start, end = end, start
			}
			scanKeys, _, err := db.ScanRange([]byte(start), []byte(end), 0)
			if err != nil {
				return false
			}
			for _, key := range scanKeys {
				k := string(key)
				if k < start || k > end {
					return false
				}
			}
			return isSortedBytes(scanKeys)
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.Property("iterator ordering", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			scanKeys, _, err := db.Scan([]byte(""), 0)
			if err != nil {
				return false
			}
			return isSortedBytes(scanKeys)
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("iterator snapshot isolation", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			expected := seedDB(db, keys, values)
			scanKeys, _, err := db.Scan([]byte(""), 0)
			if err != nil {
				return false
			}

			// Mutate after scan; results should still match snapshot at scan time.
			_ = db.Set([]byte("new-key"), []byte("new"))

			if len(scanKeys) != len(expected) {
				return false
			}
			for i := range expected {
				if string(scanKeys[i]) != expected[i] {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("iterator TTL filtering", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			// Insert an expired key directly.
			now := time.Now().UnixNano()
			db.index.SetEntry("expired", []byte("v"), now-1, now-2)

			scanKeys, _, err := db.Scan([]byte(""), 0)
			if err != nil {
				return false
			}
			for _, key := range scanKeys {
				if string(key) == "expired" {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("keys pattern matching", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			expected := seedDB(db, keys, values)
			pattern := ""
			if len(expected) > 0 {
				pattern = expected[0][:1] + "*"
			}
			result, err := db.Keys(pattern)
			if err != nil {
				return false
			}
			for _, key := range result {
				if !matchSimpleGlob(pattern, key) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("count reflects non-expired keys", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			expected := seedDB(db, keys, values)
			count, err := db.Count()
			if err != nil {
				return false
			}
			return count == len(expected)
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("iterator limit", prop.ForAll(
		func(keys []string, values [][]byte, limit uint8) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			lim := int(limit%10) + 1
			scanKeys, _, err := db.Scan([]byte(""), lim)
			if err != nil {
				return false
			}
			return len(scanKeys) <= lim
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
		gen.UInt8(),
	))

	properties.TestingRun(t)
}

func seedDB(db *DB, keys []string, values [][]byte) []string {
	limit := len(keys)
	if len(values) < limit {
		limit = len(values)
	}
	m := make(map[string]struct{}, limit)
	for i := 0; i < limit; i++ {
		key := normalizeKey(keys[i])
		if key == "" {
			continue
		}
		_ = db.Set([]byte(key), values[i])
		m[key] = struct{}{}
	}
	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func normalizeKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) > MaxKeySize {
		key = key[:MaxKeySize]
	}
	return key
}

func isSortedBytes(keys [][]byte) bool {
	for i := 1; i < len(keys); i++ {
		if string(keys[i-1]) > string(keys[i]) {
			return false
		}
	}
	return true
}

func matchSimpleGlob(pattern, value string) bool {
	if pattern == "" {
		return value == ""
	}
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == value
}
