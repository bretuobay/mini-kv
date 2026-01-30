package minikv

import (
	"testing"
)

func TestScanPrefix(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("ab"), []byte("1"))
	_ = db.Set([]byte("ac"), []byte("2"))
	_ = db.Set([]byte("ba"), []byte("3"))

	keys, values, err := db.Scan([]byte("a"), 10)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(keys) != 2 || len(values) != 2 {
		t.Fatalf("expected 2 results")
	}
	if string(keys[0]) != "ab" || string(keys[1]) != "ac" {
		t.Fatalf("unexpected keys: %q, %q", keys[0], keys[1])
	}
}

func TestScanRange(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("a"), []byte("1"))
	_ = db.Set([]byte("b"), []byte("2"))
	_ = db.Set([]byte("c"), []byte("3"))

	keys, _, err := db.ScanRange([]byte("b"), []byte("c"), 10)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 results")
	}
	if string(keys[0]) != "b" || string(keys[1]) != "c" {
		t.Fatalf("unexpected keys: %q, %q", keys[0], keys[1])
	}
}

func TestKeysPattern(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("user:1"), []byte("a"))
	_ = db.Set([]byte("user:2"), []byte("b"))
	_ = db.Set([]byte("admin:1"), []byte("c"))

	keys, err := db.Keys("user:*")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys")
	}
}

func TestCount(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("a"), []byte("1"))
	_ = db.Set([]byte("b"), []byte("2"))

	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}
