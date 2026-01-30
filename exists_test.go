package minikv

import (
	"testing"
	"time"

	"mini-kv/internal/index"
)

func TestExistsKeyTooLarge(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	key := make([]byte, db.opts.MaxKeySize+1)
	if _, err := db.Exists(key); err != ErrKeyTooLarge {
		t.Fatalf("expected ErrKeyTooLarge, got %v", err)
	}
}

func TestExistsClosed(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex(), closed: true}
	if _, err := db.Exists([]byte("k")); err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestExistsReportsState(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	key := []byte("k")

	ok, err := db.Exists(key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if ok {
		t.Fatalf("expected missing key")
	}

	db.index.SetEntry(string(key), []byte("v"), -1, time.Now().UnixNano())
	ok, err = db.Exists(key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Fatalf("expected key to exist")
	}
}
