package minikv

import (
	"testing"
)

func TestSetNX(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ok, err := db.SetNX([]byte("k"), []byte("v"))
	if err != nil || !ok {
		t.Fatalf("expected setnx success, got %v %v", ok, err)
	}
	ok, err = db.SetNX([]byte("k"), []byte("v2"))
	if err != nil || ok {
		t.Fatalf("expected setnx fail, got %v %v", ok, err)
	}
	value, _ := db.Get([]byte("k"))
	if string(value) != "v" {
		t.Fatalf("expected original value")
	}
}

func TestIncrDecr(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	val, err := db.Incr([]byte("c"))
	if err != nil || val != 1 {
		t.Fatalf("expected 1, got %d %v", val, err)
	}
	val, err = db.IncrBy([]byte("c"), 4)
	if err != nil || val != 5 {
		t.Fatalf("expected 5, got %d %v", val, err)
	}
	val, err = db.Decr([]byte("c"))
	if err != nil || val != 4 {
		t.Fatalf("expected 4, got %d %v", val, err)
	}
}

func TestCompareAndSwap(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("k"), []byte("v1"))
	ok, err := db.CompareAndSwap([]byte("k"), []byte("v1"), []byte("v2"))
	if err != nil || !ok {
		t.Fatalf("expected cas success, got %v %v", ok, err)
	}
	ok, err = db.CompareAndSwap([]byte("k"), []byte("v1"), []byte("v3"))
	if err != nil || ok {
		t.Fatalf("expected cas fail, got %v %v", ok, err)
	}
}

func TestGetAndSet(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_ = db.Set([]byte("k"), []byte("v1"))
	old, err := db.GetAndSet([]byte("k"), []byte("v2"))
	if err != nil || string(old) != "v1" {
		t.Fatalf("expected old v1, got %q %v", old, err)
	}
	value, _ := db.Get([]byte("k"))
	if string(value) != "v2" {
		t.Fatalf("expected v2, got %q", value)
	}
}
