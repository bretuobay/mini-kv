package minikv

import (
	"os"
	"sync"

	"mini-kv/internal/index"
	"mini-kv/internal/manifest"
	"mini-kv/internal/snapshot"
	"mini-kv/internal/wal"
)

// DB is the main database handle.
type DB struct {
	mu     sync.RWMutex
	path   string
	opts   Options
	index  *index.MemIndex
	wal    *wal.WALManager
	snap   *snapshot.Manager
	manifest *manifest.Manifest
	lockFile *os.File
	closed bool
}
