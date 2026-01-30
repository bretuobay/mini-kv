package minikv

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Stats provides database metrics and counters.
type Stats struct {
	KeyCount      int
	WALSize       int64
	SnapshotCount int
	MemoryBytes   int64

	Reads   uint64
	Writes  uint64
	Deletes uint64
	Scans   uint64

	ReadLatencyP50  time.Duration
	ReadLatencyP95  time.Duration
	ReadLatencyP99  time.Duration
	WriteLatencyP50 time.Duration
	WriteLatencyP95 time.Duration
	WriteLatencyP99 time.Duration
}

type statsTracker struct {
	reads   atomic.Uint64
	writes  atomic.Uint64
	deletes atomic.Uint64
	scans   atomic.Uint64

	readLatency  *latencyTracker
	writeLatency *latencyTracker
}

func newStatsTracker() *statsTracker {
	return &statsTracker{
		readLatency:  newLatencyTracker(1024),
		writeLatency: newLatencyTracker(1024),
	}
}

func (db *DB) statsOrInit() *statsTracker {
	db.statsOnce.Do(func() {
		db.stats = newStatsTracker()
	})
	return db.stats
}

type latencyTracker struct {
	mu      sync.Mutex
	samples []int64
	idx     int
	full    bool
}

func newLatencyTracker(capacity int) *latencyTracker {
	if capacity <= 0 {
		capacity = 1
	}
	return &latencyTracker{samples: make([]int64, capacity)}
}

func (l *latencyTracker) add(d time.Duration) {
	l.mu.Lock()
	l.samples[l.idx] = d.Nanoseconds()
	l.idx++
	if l.idx >= len(l.samples) {
		l.idx = 0
		l.full = true
	}
	l.mu.Unlock()
}

func (l *latencyTracker) percentiles() (time.Duration, time.Duration, time.Duration) {
	l.mu.Lock()
	count := l.idx
	if l.full {
		count = len(l.samples)
	}
	if count == 0 {
		l.mu.Unlock()
		return 0, 0, 0
	}
	snapshot := make([]int64, count)
	copy(snapshot, l.samples[:count])
	l.mu.Unlock()

	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i] < snapshot[j] })
	return percentile(snapshot, 0.50), percentile(snapshot, 0.95), percentile(snapshot, 0.99)
}

func percentile(values []int64, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	idx := int(float64(len(values)-1) * p)
	return time.Duration(values[idx]) * time.Nanosecond
}

// Stats returns current metrics.
func (db *DB) Stats() (Stats, error) {
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return Stats{}, ErrClosed
	}
	statsTracker := db.statsOrInit()
	keyCount := db.index.Count()
	memBytes := db.index.Size()
	walDir := filepath.Join(db.path, "wal")
	snapDir := filepath.Join(db.path, "snapshots")
	db.mu.RUnlock()

	walSize := dirSize(walDir, ".log")
	snapCount := dirCount(snapDir, ".snap")

	readP50, readP95, readP99 := statsTracker.readLatency.percentiles()
	writeP50, writeP95, writeP99 := statsTracker.writeLatency.percentiles()

	return Stats{
		KeyCount:        keyCount,
		WALSize:         walSize,
		SnapshotCount:   snapCount,
		MemoryBytes:     memBytes,
		Reads:           statsTracker.reads.Load(),
		Writes:          statsTracker.writes.Load(),
		Deletes:         statsTracker.deletes.Load(),
		Scans:           statsTracker.scans.Load(),
		ReadLatencyP50:  readP50,
		ReadLatencyP95:  readP95,
		ReadLatencyP99:  readP99,
		WriteLatencyP50: writeP50,
		WriteLatencyP95: writeP95,
		WriteLatencyP99: writeP99,
	}, nil
}

// DumpKeys writes all non-expired keys with metadata to w.
func (db *DB) DumpKeys(w io.Writer) error {
	db.mu.RLock()
	if db.closed {
		db.mu.RUnlock()
		return ErrClosed
	}
	entries := db.index.Scan("", 0)
	db.mu.RUnlock()

	writer := bufio.NewWriter(w)
	for _, entry := range entries {
		expires := "-1"
		if entry.Entry.ExpiresAt >= 0 {
			expires = time.Unix(0, entry.Entry.ExpiresAt).UTC().Format(time.RFC3339Nano)
		}
		if _, err := writer.WriteString(string(entry.Key)); err != nil {
			return err
		}
		if _, err := writer.WriteString("\t"); err != nil {
			return err
		}
		if _, err := writer.WriteString(intToString(len(entry.Entry.Value))); err != nil {
			return err
		}
		if _, err := writer.WriteString("\t"); err != nil {
			return err
		}
		if _, err := writer.WriteString(expires); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func dirSize(path string, suffix string) int64 {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return total
}

func dirCount(path string, suffix string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), suffix) {
			count++
		}
	}
	return count
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		digit := v % 10
		buf = append(buf, byte('0'+digit))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
