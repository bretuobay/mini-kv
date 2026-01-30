package minikv

import "testing"

func TestCrashRecoverySimulation(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultOptions(dir)

	db, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Set([]byte("alpha"), []byte("1")); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Simulate crash by closing file handle without Close (release lock).
	_ = db.lockFile.Close()
	db.lockFile = nil

	db2, err := Open(opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	value, err := db2.Get([]byte("alpha"))
	if err != nil || string(value) != "1" {
		t.Fatalf("expected recovered value, got %v %v", value, err)
	}
}
