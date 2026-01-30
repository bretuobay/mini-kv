package minikv

import (
	"testing"
	"time"
)

func TestBatchWriteAppliesOperations(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()
	batch.Set([]byte("a"), []byte("1"))
	batch.SetWithTTL([]byte("b"), []byte("2"), 50*time.Millisecond)
	batch.Delete([]byte("c"))

	if err := batch.Write(); err != nil {
		t.Fatalf("write: %v", err)
	}

	if value, err := db.Get([]byte("a")); err != nil || string(value) != "1" {
		t.Fatalf("expected a=1, got %v %v", value, err)
	}
	if _, err := db.Get([]byte("b")); err != nil {
		t.Fatalf("expected b to exist, got %v", err)
	}
}

func TestBatchDiscard(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()
	batch.Set([]byte("a"), []byte("1"))
	batch.Discard()

	if err := batch.Write(); err != ErrClosed {
		t.Fatalf("expected ErrClosed after discard, got %v", err)
	}
}

func TestBatchTooBig(t *testing.T) {
	opts := DefaultOptions(t.TempDir())
	opts.MaxBatchSize = 10
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()
	batch.Set([]byte("a"), []byte("12345678901"))

	if err := batch.Write(); err != ErrBatchTooBig {
		t.Fatalf("expected ErrBatchTooBig, got %v", err)
	}
}

func TestBatchValidatesSizes(t *testing.T) {
	opts := DefaultOptions(t.TempDir())
	opts.MaxKeySize = 2
	opts.MaxValueSize = 2
	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()
	batch.Set([]byte("toolong"), []byte("v"))
	if err := batch.Write(); err != ErrKeyTooLarge {
		t.Fatalf("expected ErrKeyTooLarge, got %v", err)
	}
}
