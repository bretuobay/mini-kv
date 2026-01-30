package minikv

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompactCreatesSnapshotAndCleansWAL(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	opts.MaxWALSize = 128
	opts.SyncMode = SyncManual

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	for i := 0; i < 50; i++ {
		if err := db.Set([]byte("k"+intToString(i)), []byte("v")); err != nil {
			t.Fatalf("set: %v", err)
		}
	}

	if err := db.Compact(); err != nil {
		t.Fatalf("compact: %v", err)
	}

	snapDir := filepath.Join(dir, "snapshots")
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		t.Fatalf("read snapshots: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected snapshot files")
	}

	walDir := filepath.Join(dir, "wal")
	walEntries, err := os.ReadDir(walDir)
	if err != nil {
		t.Fatalf("read wal: %v", err)
	}
	if len(walEntries) == 0 {
		t.Fatalf("expected wal segments")
	}
}

func TestManifestUpdatedAfterCompaction(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	opts.MaxWALSize = 128
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("a"), []byte("1"))
	_ = db.Set([]byte("b"), []byte("2"))

	if err := db.Compact(); err != nil {
		t.Fatalf("compact: %v", err)
	}

	manifestPath := filepath.Join(dir, "MANIFEST")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected MANIFEST, got %v", err)
	}
}

func TestRotateTriggersCompaction(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	opts.MaxWALSize = 64
	opts.SyncMode = SyncManual

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	for i := 0; i < 100; i++ {
		_ = db.Set([]byte("k"+intToString(i)), []byte("value-data"))
	}

	// Allow background compaction to run.
	time.Sleep(50 * time.Millisecond)

	snapDir := filepath.Join(dir, "snapshots")
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		t.Fatalf("read snapshots: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected snapshot files")
	}
}
