package minikv

import (
	"sync"
	"testing"
)

func TestFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Set([]byte("alpha"), []byte("1")); err != nil {
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

	value, err := db2.Get([]byte("alpha"))
	if err != nil || string(value) != "1" {
		t.Fatalf("expected persisted value, got %v %v", value, err)
	}
}

func TestConcurrentOperations(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := []byte("k" + intToString(id))
			_ = db.Set(key, []byte("v"))
			_, _ = db.Get(key)
			_, _ = db.Exists(key)
		}(i)
	}
	wg.Wait()

	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected some keys")
	}
}
