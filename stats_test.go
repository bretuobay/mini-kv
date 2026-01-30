package minikv

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatsReflectsOps(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("a"), []byte("1"))
	_, _ = db.Get([]byte("a"))
	_, _ = db.Exists([]byte("a"))
	_, _, _ = db.Scan([]byte("a"), 10)

	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Writes == 0 || stats.Reads == 0 || stats.Scans == 0 {
		t.Fatalf("expected counters to be incremented, got %+v", stats)
	}
}

func TestDumpKeysWritesOutput(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("a"), []byte("1"))
	_ = db.Set([]byte("b"), []byte("2"))

	var buf bytes.Buffer
	if err := db.DumpKeys(&buf); err != nil {
		t.Fatalf("dump: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a\t") || !strings.Contains(out, "b\t") {
		t.Fatalf("expected output to include keys, got %q", out)
	}
}
