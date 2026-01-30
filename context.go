package minikv

import "context"

// GetWithContext returns the value for a key honoring context cancellation.
func (db *DB) GetWithContext(ctx context.Context, key []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return db.Get(key)
}

// SetWithContext stores a key-value pair honoring context cancellation.
func (db *DB) SetWithContext(ctx context.Context, key []byte, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return db.Set(key, value)
}
