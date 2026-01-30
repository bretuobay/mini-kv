package minikv

import "time"

func (db *DB) startTTLWorker() {
	if db.stopCh == nil {
		db.stopCh = make(chan struct{})
	}
	if db.ttlTicker == nil {
		db.ttlTicker = time.NewTicker(1 * time.Second)
	}

	db.wg.Add(1)
	go func() {
		defer db.wg.Done()
		for {
			select {
			case <-db.ttlTicker.C:
				db.cleanupExpired()
			case <-db.stopCh:
				return
			}
		}
	}()
}

func (db *DB) cleanupExpired() {
	db.mu.Lock()
	if db.closed {
		db.mu.Unlock()
		return
	}
	// Count() performs expiration cleanup under the index lock; avoid holding DB lock.
	db.mu.Unlock()
	_ = db.index.Count()
}
