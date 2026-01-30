package minikv

import (
	"bytes"
	"testing"
	"time"

	"mini-kv/internal/index"
)

func TestGetKeyTooLarge(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp")}
	key := make([]byte, db.opts.MaxKeySize+1)
	if _, err := db.Get(key); err != ErrKeyTooLarge {
		t.Fatalf("expected ErrKeyTooLarge, got %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	if _, err := db.Get([]byte("missing")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetReturnsValue(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	key := []byte("hello")
	value := []byte("world")
	db.index.SetEntry(string(key), value, -1, time.Now().UnixNano())
	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("expected %q, got %q", value, got)
	}
	got[0] = 'x'
	got2, _ := db.Get(key)
	if bytes.Equal(got, got2) {
		t.Fatalf("expected value copy")
	}
}
