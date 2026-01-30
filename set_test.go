package minikv

import (
	"testing"

	"github.com/bretuobay/mini-kv/internal/index"
)

func TestSetValidatesKeySize(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	key := make([]byte, db.opts.MaxKeySize+1)
	if err := db.Set(key, []byte("v")); err != ErrKeyTooLarge {
		t.Fatalf("expected ErrKeyTooLarge, got %v", err)
	}
}

func TestSetValidatesValueSize(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	value := make([]byte, db.opts.MaxValueSize+1)
	if err := db.Set([]byte("k"), value); err != ErrValueTooLarge {
		t.Fatalf("expected ErrValueTooLarge, got %v", err)
	}
}

func TestSetReadOnly(t *testing.T) {
	db := &DB{opts: Options{ReadOnly: true, MaxKeySize: MaxKeySize, MaxValueSize: MaxValueSize}, index: index.NewMemIndex()}
	if err := db.Set([]byte("k"), []byte("v")); err != ErrReadOnly {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestSetClosed(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex(), closed: true}
	if err := db.Set([]byte("k"), []byte("v")); err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}
