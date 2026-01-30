package minikv

import (
	"testing"
)

func TestSyncManualDoesNotStartWorker(t *testing.T) {
	opts := DefaultOptions(t.TempDir())
	opts.SyncMode = SyncManual

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if db.syncTicker != nil {
		t.Fatalf("expected no sync ticker for manual mode")
	}
}

func TestSyncPeriodicStartsWorker(t *testing.T) {
	opts := DefaultOptions(t.TempDir())
	opts.SyncMode = SyncPeriodic

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if db.syncTicker == nil {
		t.Fatalf("expected sync ticker for periodic mode")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if db.syncTicker != nil || db.stopCh != nil {
		t.Fatalf("expected workers stopped on close")
	}
}

func TestSyncClosed(t *testing.T) {
	db := &DB{closed: true}
	if err := db.Sync(); err != ErrClosed {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}
