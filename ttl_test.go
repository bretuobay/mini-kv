package minikv

import (
	"testing"
	"time"
)

func TestSetWithTTLExpires(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(DefaultOptions(dir))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.SetWithTTL([]byte("k"), []byte("v"), 10*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := db.Get([]byte("k")); err != nil {
		t.Fatalf("get: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	if _, err := db.Get([]byte("k")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTTLReturnsRemaining(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(DefaultOptions(dir))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.SetWithTTL([]byte("k"), []byte("v"), 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	remaining, err := db.TTL([]byte("k"))
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if remaining <= 0 {
		t.Fatalf("expected remaining ttl, got %v", remaining)
	}
}

func TestTTLNoExpiration(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(DefaultOptions(dir))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Set([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("set: %v", err)
	}
	remaining, err := db.TTL([]byte("k"))
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if remaining != -1 {
		t.Fatalf("expected -1, got %v", remaining)
	}
}

func TestExpireAndPersist(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(DefaultOptions(dir))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Set([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("set: %v", err)
	}
	ok, err := db.Expire([]byte("k"), 50*time.Millisecond)
	if err != nil || !ok {
		t.Fatalf("expire: %v, %v", ok, err)
	}
	ok, err = db.Persist([]byte("k"))
	if err != nil || !ok {
		t.Fatalf("persist: %v, %v", ok, err)
	}
	remaining, err := db.TTL([]byte("k"))
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if remaining != -1 {
		t.Fatalf("expected -1, got %v", remaining)
	}
}
