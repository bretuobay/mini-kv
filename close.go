package minikv

import "syscall"

// Close flushes pending work and releases resources.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}
	db.closed = true

	var err error
	if db.wal != nil {
		if syncErr := db.wal.Sync(); syncErr != nil {
			err = syncErr
		}
		if closeErr := db.wal.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	if db.lockFile != nil {
		_ = syscall.Flock(int(db.lockFile.Fd()), syscall.LOCK_UN)
		if closeErr := db.lockFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		db.lockFile = nil
	}

	return err
}
