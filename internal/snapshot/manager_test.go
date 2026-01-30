package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestSnapshotManagerExcludesExpiredKeys(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("manager excludes expired keys", prop.ForAll(
		func(entries []Entry, timestamp int64) bool {
			if timestamp < 0 {
				timestamp = -timestamp
			}
			manager := NewManager(t.TempDir())
			path, err := manager.CreateSnapshot(entries, 1, timestamp)
			if err != nil {
				return false
			}

			_, decoded, err := manager.LoadSnapshot(path)
			if err != nil {
				return false
			}

			for _, entry := range decoded {
				if entry.ExpiresAt >= 0 && entry.ExpiresAt <= timestamp {
					return false
				}
			}
			return true
		},
		gen.SliceOf(genEntry()),
		gen.Int64(),
	))

	properties.TestingRun(t)
}

func TestSnapshotManagerCreatesFile(t *testing.T) {
	manager := NewManager(t.TempDir())
	path, err := manager.CreateSnapshot([]Entry{{Key: []byte("a"), Value: []byte("b")}}, 1, 1)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot file, got error: %v", err)
	}
	if filepath.Ext(path) != ".snap" {
		t.Fatalf("expected .snap file, got %s", path)
	}
}
