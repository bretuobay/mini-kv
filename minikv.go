package minikv

import (
	"os"
	"sync"
	"time"

	"mini-kv/internal/index"
	"mini-kv/internal/manifest"
	"mini-kv/internal/snapshot"
	"mini-kv/internal/wal"
)

// DB is the main database handle.
type DB struct {
	mu         sync.RWMutex
	path       string
	opts       Options
	index      *index.MemIndex
	wal        *wal.WALManager
	snap       *snapshot.Manager
	manifest   *manifest.Manifest
	lockFile   *os.File
	syncTicker *time.Ticker
	ttlTicker  *time.Ticker
	stats      *statsTracker
	statsOnce  sync.Once
	compactMu  sync.Mutex
	compacting bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	closed     bool
}
