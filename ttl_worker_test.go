package minikv

import "testing"

func TestTTLWorkerStartsAndStops(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if db.ttlTicker == nil {
		t.Fatalf("expected ttl ticker to start")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if db.ttlTicker != nil || db.stopCh != nil {
		t.Fatalf("expected ttl worker stopped")
	}
}
