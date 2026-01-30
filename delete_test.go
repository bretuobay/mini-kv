package minikv

import (
	"testing"

	"github.com/bretuobay/mini-kv/internal/index"
)

func TestDeleteReadOnly(t *testing.T) {
	db := &DB{opts: Options{ReadOnly: true, MaxKeySize: MaxKeySize, MaxValueSize: MaxValueSize}, index: index.NewMemIndex()}
	if err := db.Delete([]byte("k")); err != ErrReadOnly {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
}

func TestDeleteClosed(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex(), closed: true}
	if err := db.Delete([]byte("k")); err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestDeleteKeyTooLarge(t *testing.T) {
	db := &DB{opts: DefaultOptions("/tmp"), index: index.NewMemIndex()}
	key := make([]byte, db.opts.MaxKeySize+1)
	if err := db.Delete(key); err != ErrKeyTooLarge {
		t.Fatalf("expected ErrKeyTooLarge, got %v", err)
	}
}
