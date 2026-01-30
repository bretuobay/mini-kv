package minikv

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/bretuobay/mini-kv/internal/wal"
)

func TestOpenCloseReopenPreservesData(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("open-close-reopen preserves data", prop.ForAll(
		func(key string, value []byte) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)

			db, err := Open(opts)
			if err != nil {
				return false
			}

			// Manually persist via WAL to simulate a basic write in absence of Set.
			record := wal.WALRecord{
				Type:      wal.RecordSet,
				Timestamp: 1,
				ExpiresAt: -1,
				Key:       []byte(key),
				Value:     value,
			}
			if err := db.wal.AppendRecord(record); err != nil {
				_ = db.Close()
				return false
			}
			db.index.SetEntry(key, value, -1, 1)

			if err := db.Close(); err != nil {
				return false
			}

			db2, err := Open(opts)
			if err != nil {
				return false
			}
			defer db2.Close()

			entry, ok := db2.index.Get(key)
			if !ok {
				return false
			}
			return bytes.Equal(entry.Value, value)
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

func TestExclusiveFileLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	db, err := Open(DefaultOptions(dir))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := Open(DefaultOptions(dir)); err == nil {
		t.Fatalf("expected lock error")
	}
}

func TestCloseReleasesLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	opts := DefaultOptions(path)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := Open(opts); err != nil {
		t.Fatalf("expected reopen to succeed, got %v", err)
	}
}
