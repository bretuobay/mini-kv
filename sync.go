package minikv

import "time"

// Sync flushes WAL data to disk when SyncMode is manual.
func (db *DB) Sync() error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return ErrClosed
	}
	if db.wal == nil {
		return nil
	}
	return db.wal.Sync()
}

func (db *DB) startSyncWorker() {
	if db.opts.SyncMode != SyncPeriodic {
		return
	}
	if db.stopCh == nil {
		db.stopCh = make(chan struct{})
	}
	if db.syncTicker == nil {
		db.syncTicker = time.NewTicker(1 * time.Second)
	}

	db.wg.Add(1)
	go func() {
		defer db.wg.Done()
		for {
			select {
			case <-db.syncTicker.C:
				_ = db.Sync()
			case <-db.stopCh:
				return
			}
		}
	}()
}

func (db *DB) stopWorkers() {
	if db.syncTicker != nil {
		db.syncTicker.Stop()
	}
	if db.ttlTicker != nil {
		db.ttlTicker.Stop()
	}
	if db.stopCh != nil {
		close(db.stopCh)
	}
	db.wg.Wait()
	db.syncTicker = nil
	db.ttlTicker = nil
	db.stopCh = nil
}
