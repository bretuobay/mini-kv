package minikv

import (
	"os"
	"sync"
	"time"

	"github.com/bretuobay/mini-kv/internal/index"
	"github.com/bretuobay/mini-kv/internal/manifest"
	"github.com/bretuobay/mini-kv/internal/snapshot"
	"github.com/bretuobay/mini-kv/internal/wal"
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
