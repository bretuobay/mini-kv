package minikv

// Set stores a key-value pair.
func (db *DB) Set(key []byte, value []byte) error {
	return db.setWithExpiresAt(key, value, -1)
}
