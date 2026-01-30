package minikv

import (
	"testing"
)

func TestLargeDatasetIntegration(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)
	opts.SyncMode = SyncManual
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	for i := 0; i < 5000; i++ {
		if err := db.Set([]byte("key-"+intToString(i)), []byte("value")); err != nil {
			t.Fatalf("set: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	count, err := db2.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 5000 {
		t.Fatalf("expected 5000 keys, got %d", count)
	}
}
