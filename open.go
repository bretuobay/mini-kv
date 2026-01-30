package minikv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mini-kv/internal/index"
	"mini-kv/internal/manifest"
	"mini-kv/internal/snapshot"
	"mini-kv/internal/wal"
)

// Open opens or creates a database at the given path.
func Open(opts Options) (*DB, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return nil, fmt.Errorf("minikv: path required")
	}
	opts = withDefaults(opts)

	if err := os.MkdirAll(opts.Path, 0o755); err != nil {
		return nil, err
	}

	lockFile, err := os.OpenFile(filepath.Join(opts.Path, "LOCK"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lockFile.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}

	idx := index.NewMemIndex()
	snapMgr := snapshot.NewManager(filepath.Join(opts.Path, "snapshots"))
	walMgr, err := wal.OpenWAL(filepath.Join(opts.Path, "wal"), opts.MaxWALSize)
	if err != nil {
		_ = lockFile.Close()
		return nil, err
	}

	manifestPath := filepath.Join(opts.Path, "MANIFEST")
	man, err := loadManifest(manifestPath)
	if err != nil {
		_ = walMgr.Close()
		_ = lockFile.Close()
		return nil, err
	}

	if path, ok := latestSnapshotPath(man); ok {
		head, entries, err := snapMgr.LoadSnapshot(path)
		if err != nil {
			_ = walMgr.Close()
			_ = lockFile.Close()
			return nil, err
		}
		_ = head
		now := time.Now().UnixNano()
		for _, entry := range entries {
			if entry.ExpiresAt >= 0 && entry.ExpiresAt <= now {
				continue
			}
			idx.SetEntry(string(entry.Key), entry.Value, entry.ExpiresAt, entry.CreatedAt)
		}
	}

	if err := replayWAL(idx, filepath.Join(opts.Path, "wal"), man.LastSnapshotSeq); err != nil {
		_ = walMgr.Close()
		_ = lockFile.Close()
		return nil, err
	}

	_ = refreshManifest(opts.Path)

	db := &DB{
		path:     opts.Path,
		opts:     opts,
		index:    idx,
		wal:      walMgr,
		snap:     snapMgr,
		manifest: &man,
		lockFile: lockFile,
		stats:    newStatsTracker(),
	}
	walMgr.SetRotateHook(func() {
		db.compactAsync()
		_ = refreshManifest(opts.Path)
	})
	db.startSyncWorker()
	db.startTTLWorker()
	return db, nil
}

func withDefaults(opts Options) Options {
	if opts.MaxKeySize == 0 {
		opts.MaxKeySize = MaxKeySize
	}
	if opts.MaxValueSize == 0 {
		opts.MaxValueSize = MaxValueSize
	}
	if opts.MaxBatchSize == 0 {
		opts.MaxBatchSize = MaxBatchSize
	}
	if opts.MaxWALSize == 0 {
		opts.MaxWALSize = MaxWALSize
	}
	if opts.SyncMode == 0 {
		opts.SyncMode = SyncPeriodic
	}
	return opts
}

func loadManifest(path string) (manifest.Manifest, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return manifest.Manifest{}, nil
		}
		return manifest.Manifest{}, err
	}
	return manifest.ReadManifest(path)
}

func latestSnapshotPath(man manifest.Manifest) (string, bool) {
	if len(man.Snapshots) == 0 {
		return "", false
	}
	var latest manifest.SnapshotInfo
	for i, snap := range man.Snapshots {
		if i == 0 || snap.Seq > latest.Seq {
			latest = snap
		}
	}
	if latest.Path == "" {
		return "", false
	}
	return latest.Path, true
}

func replayWAL(idx *index.MemIndex, walDir string, minSeq uint64) error {
	segments, err := wal.ListSegments(walDir)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	for _, path := range segments {
		seq, ok := parseSegmentSeq(path)
		if ok && seq <= minSeq {
			continue
		}
		records, err := wal.ReadWAL(path)
		if err != nil {
			return err
		}
		for _, rec := range records {
			switch rec.Type {
			case wal.RecordDelete:
				idx.Delete(string(rec.Key))
			case wal.RecordSet:
				if rec.ExpiresAt >= 0 && rec.ExpiresAt <= now {
					idx.Delete(string(rec.Key))
					continue
				}
				idx.SetEntry(string(rec.Key), rec.Value, rec.ExpiresAt, rec.Timestamp)
			}
		}
	}
	return nil
}

func parseSegmentSeq(path string) (uint64, bool) {
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".log") {
		return 0, false
	}
	value := strings.TrimSuffix(base, ".log")
	seq, err := parseUint(value)
	if err != nil {
		return 0, false
	}
	return seq, true
}

func parseUint(value string) (uint64, error) {
	var seq uint64
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		seq = seq*10 + uint64(ch-'0')
	}
	return seq, nil
}
