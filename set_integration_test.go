package minikv

import (
	"testing"
)

func TestSetWritesWAL(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Set([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	value, err := db2.Get([]byte("key"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(value) != "value" {
		t.Fatalf("expected value, got %s", value)
	}
}
