package snapshot

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestSnapshotRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("snapshot encode/decode round trip", prop.ForAll(
		func(entries []Entry, version uint32, timestamp int64) bool {
			dir := t.TempDir()
			path := filepath.Join(dir, "snapshot.snap")
			file, err := os.Create(path)
			if err != nil {
				return false
			}

			_, err = EncodeSnapshot(file, entries, version, timestamp)
			_ = file.Close()
			if err != nil {
				return false
			}

			head, decoded, err := DecodeSnapshot(path)
			if err != nil {
				return false
			}
			if head.Version != version || head.Timestamp != timestamp || int(head.Count) != len(entries) {
				return false
			}
			// EncodeSnapshot sorts entries by key; ensure decoded matches sorted original.
			sorted := sortEntries(entries)
			if len(sorted) != len(decoded) {
				return false
			}
			for i := range sorted {
				if !bytes.Equal(sorted[i].Key, decoded[i].Key) {
					return false
				}
				if !bytes.Equal(sorted[i].Value, decoded[i].Value) {
					return false
				}
				if sorted[i].ExpiresAt != decoded[i].ExpiresAt || sorted[i].CreatedAt != decoded[i].CreatedAt {
					return false
				}
			}
			return true
		},
		gen.SliceOf(genEntry()),
		gen.UInt32(),
		gen.Int64(),
	))

	properties.TestingRun(t)
}

// Expired key exclusion is verified via SnapshotManager tests.

func genEntry() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOf(gen.UInt8()),
		gen.SliceOf(gen.UInt8()),
		gen.Int64(),
		gen.Int64(),
	).Map(func(values []interface{}) Entry {
		key := values[0].([]byte)
		if len(key) == 0 {
			key = []byte{0}
		}
		return Entry{
			Key:       key,
			Value:     values[1].([]byte),
			ExpiresAt: values[2].(int64),
			CreatedAt: values[3].(int64),
		}
	})
}

func sortEntries(entries []Entry) []Entry {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return string(sorted[i].Key) < string(sorted[j].Key)
	})
	return sorted
}
