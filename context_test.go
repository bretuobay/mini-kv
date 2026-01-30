package minikv

import (
	"context"
	"testing"
)

func TestContextCancellation(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.GetWithContext(ctx, []byte("k")); err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if err := db.SetWithContext(ctx, []byte("k"), []byte("v")); err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
