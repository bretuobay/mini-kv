package minikv

import "testing"

func TestDeleteRemovesValue(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Set([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := db.Delete([]byte("key")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.Get([]byte("key")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	if _, err := db2.Get([]byte("key")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after reopen, got %v", err)
	}
}
